use std::sync::Arc;
use tokio::sync::Mutex;
use tonic::{transport::Server, Request, Response, Status};

use candle_core::{Device, Tensor, DType};
use candle_transformers::models::bert::{BertModel, Config as BertConfig};
use candle_nn::VarBuilder;
use tokenizers::Tokenizer;
use hf_hub::{api::sync::Api, Repo, RepoType};

// Generated proto code
pub mod sidecar {
    tonic::include_proto!("sidecar");
}

use sidecar::{llm_service_server::{LlmService, LlmServiceServer}, *};

// Real embedding model using candle
struct EmbeddingModel {
    model: Option<BertModel>,
    tokenizer: Option<Tokenizer>,
    device: Device,
    model_path: String,
    embedding_dim: usize,
}

impl EmbeddingModel {
    fn new() -> Self {
        Self {
            model: None,
            tokenizer: None,
            device: Device::Cpu,
            model_path: String::new(),
            embedding_dim: 384,
        }
    }

    fn load(&mut self, model_path: &str) -> anyhow::Result<()> {
        tracing::info!("Loading embedding model from: {}", model_path);

        // Check if path is a HuggingFace model ID or local path
        let (tokenizer, config_filename, weights_filename) = if model_path.contains('/') {
            // HuggingFace model ID
            tracing::info!("Downloading model from HuggingFace: {}", model_path);
            let api = Api::new()?;
            let repo = Repo::with_revision(model_path.to_string(), RepoType::Model, "main".to_string());
            let api = api.repo(repo);

            let tokenizer_path = api.get("tokenizer.json")?;
            let config_path = api.get("config.json")?;
            let model_path = api.get("model.safetensors")?;

            let tokenizer = Tokenizer::from_file(tokenizer_path).map_err(|e| anyhow::anyhow!("{}", e))?;
            (tokenizer, config_path.to_string_lossy().to_string(), model_path.to_string_lossy().to_string())
        } else {
            // Local path
            tracing::info!("Loading model from local path: {}", model_path);
            let base_path = std::path::Path::new(model_path);
            let tokenizer_path = base_path.join("tokenizer.json");
            let config_path = base_path.join("config.json");
            let model_path = base_path.join("model.safetensors");

            if !tokenizer_path.exists() {
                anyhow::bail!("tokenizer.json not found in {}", base_path.display());
            }
            if !config_path.exists() {
                anyhow::bail!("config.json not found in {}", base_path.display());
            }
            if !model_path.exists() {
                anyhow::bail!("model.safetensors not found in {}", base_path.display());
            }

            let tokenizer = Tokenizer::from_file(&tokenizer_path).map_err(|e| anyhow::anyhow!("{}", e))?;
            (tokenizer, config_path.to_string_lossy().to_string(), model_path.to_string_lossy().to_string())
        };

        // Load config
        let config = std::fs::read_to_string(&config_filename)?;
        let config: BertConfig = serde_json::from_str(&config)?;
        self.embedding_dim = config.hidden_size;

        tracing::info!("Model config: hidden_size={}, num_layers={}", config.hidden_size, config.num_hidden_layers);

        // Load model
        let vb = unsafe {
            VarBuilder::from_mmaped_safetensors(&[&weights_filename], DType::F32, &self.device)?
        };
        let model = BertModel::load(vb, &config)?;

        self.model = Some(model);
        self.tokenizer = Some(tokenizer);
        self.model_path = model_path.to_string();

        tracing::info!("Embedding model loaded successfully");
        Ok(())
    }

    fn embed(&self, text: &str) -> anyhow::Result<Vec<f32>> {
        let model = self.model.as_ref().ok_or(anyhow::anyhow!("Model not loaded"))?;
        let tokenizer = self.tokenizer.as_ref().ok_or(anyhow::anyhow!("Tokenizer not loaded"))?;

        // Tokenize input
        let tokens = tokenizer
            .encode(text, true)
            .map_err(|e| anyhow::anyhow!("Tokenization failed: {}", e))?;

        let input_ids = Tensor::new(
            tokens.get_ids().iter().map(|&i| i as i64).collect::<Vec<_>>(),
            &self.device,
        )?
        .unsqueeze(0)?;

        let attention_mask = Tensor::new(
            tokens.get_attention_mask().iter().map(|&i| i as u8).collect::<Vec<_>>(),
            &self.device,
        )?
        .unsqueeze(0)?;

        // Generate embeddings
        let embeddings = model.forward(&input_ids, &attention_mask, None)?;

        // Mean pooling (average all token embeddings)
        let embeddings = embeddings.mean(1)?;

        // Squeeze batch dimension and convert to Vec<f32>
        let result = embeddings.squeeze(0)?.to_vec1::<f32>()?;
        Ok(result)
    }
}

// Service implementation
struct LLMServiceImpl {
    model: Arc<Mutex<EmbeddingModel>>,
}

impl Default for LLMServiceImpl {
    fn default() -> Self {
        Self {
            model: Arc::new(Mutex::new(EmbeddingModel::new())),
        }
    }
}

#[tonic::async_trait]
impl LlmService for LLMServiceImpl {
    async fn init_model(&self, request: Request<InitRequest>) -> Result<Response<InitResponse>, Status> {
        let req = request.into_inner();
        let mut model = self.model.lock().await;

        match model.load(&req.model_path) {
            Ok(_) => Ok(Response::new(InitResponse {
                success: true,
                message: format!("Embedding model loaded from {}", req.model_path),
            })),
            Err(e) => Ok(Response::new(InitResponse {
                success: false,
                message: format!("Failed to load model: {}", e),
            })),
        }
    }

    type GenerateStream = tokio_stream::wrappers::ReceiverStream<Result<GenerateResponse, Status>>;

    async fn generate(&self, request: Request<GenerateRequest>) -> Result<Response<Self::GenerateStream>, Status> {
        let req = request.into_inner();
        let model = self.model.lock().await;

        let (tx, rx) = tokio::sync::mpsc::channel(4);

        let prompt = req.prompt;
        let is_loaded = model.model.is_some();

        let embedding_result: anyhow::Result<Vec<f32>> = if is_loaded {
            model.embed(&prompt)
        } else {
            Ok(vec![])
        };

        drop(model);

        tokio::spawn(async move {
            if !is_loaded {
                let _ = tx.send(Err(Status::failed_precondition("Model not initialized"))).await;
                return;
            }

            match embedding_result {
                Ok(embedding) => {
                    if embedding.is_empty() {
                        let _ = tx.send(Err(Status::internal("Failed to generate embedding"))).await;
                        return;
                    }

                    // Show embedding info
                    let non_zero_count = embedding.iter().filter(|&&x| x.abs() > 1e-6).count();
                    let max_val = embedding.iter().fold(0.0f32, |a, &b| a.max(b.abs()));
                    let min_val = embedding.iter().fold(0.0f32, |a, &b| a.min(b.abs()));

                    let header = format!(
                        "Embedding generated for: '{}'\nDim: {} | Non-zero: {} | Range: [{:.4}, {:.4}]\n\nVector (hex):\n",
                        prompt,
                        embedding.len(),
                        non_zero_count,
                        min_val,
                        max_val
                    );

                    // Send header
                    for ch in header.chars() {
                        if tx.send(Ok(GenerateResponse {
                            text: ch.to_string(),
                            done: false,
                            tokens_generated: 0,
                        })).await.is_err() {
                            return;
                        }
                        tokio::time::sleep(tokio::time::Duration::from_millis(1)).await;
                    }

                    // Send embedding vector in hex format (8 values per line)
                    for (i, val) in embedding.iter().enumerate() {
                        let hex_val = format!("{:08x}", val.to_bits());
                        let comma = if (i + 1) % 8 == 0 { "\n" } else { " " };
                        let text = format!("{}{}", hex_val, comma);

                        if tx.send(Ok(GenerateResponse {
                            text,
                            done: false,
                            tokens_generated: i as i32,
                        })).await.is_err() {
                            return;
                        }
                        tokio::time::sleep(tokio::time::Duration::from_millis(1)).await;
                    }

                    // Send done signal
                    let _ = tx.send(Ok(GenerateResponse {
                        text: String::new(),
                        done: true,
                        tokens_generated: embedding.len() as i32,
                    })).await;
                }
                Err(e) => {
                    let _ = tx.send(Err(Status::internal(format!("Embedding error: {}", e)))).await;
                }
            }
        });

        Ok(Response::new(tokio_stream::wrappers::ReceiverStream::new(rx)))
    }

    async fn embed(&self, request: Request<EmbedRequest>) -> Result<Response<EmbedResponse>, Status> {
        let req = request.into_inner();
        let model = self.model.lock().await;

        let result = if model.model.is_some() {
            match model.embed(&req.text) {
                Ok(vector) => Ok(EmbedResponse {
                    vector,
                    dim: model.embedding_dim as i32,
                }),
                Err(e) => Err(Status::internal(format!("Embedding error: {}", e))),
            }
        } else {
            Err(Status::failed_precondition("Model not initialized"))
        };

        result.map(Response::new)
    }

    async fn model_info(&self, _request: Request<ModelInfoRequest>) -> Result<Response<ModelInfoResponse>, Status> {
        let model = self.model.lock().await;
        Ok(Response::new(ModelInfoResponse {
            model_name: if model.model.is_some() {
                format!("{} (candle BERT)", model.model_path)
            } else {
                "Not loaded".to_string()
            },
            vocab_size: 30522,
            context_size: 512,
            backend: "candle".to_string(),
        }))
    }

    async fn health(&self, _request: Request<HealthRequest>) -> Result<Response<HealthResponse>, Status> {
        let model = self.model.lock().await;
        Ok(Response::new(HealthResponse {
            healthy: true,
            message: if model.model.is_some() {
                "Embedding service is healthy (model loaded)".to_string()
            } else {
                "Embedding service is healthy (no model)".to_string()
            },
        }))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .init();

    let addr = "[::0]:50051".parse()?;
    let llm_service = LLMServiceImpl::default();

    tracing::info!("LLM Embedding Sidecar listening on {}", addr);
    tracing::info!("Using candle for real BERT embedding models");

    Server::builder()
        .add_service(LlmServiceServer::new(llm_service))
        .serve(addr)
        .await?;

    Ok(())
}
