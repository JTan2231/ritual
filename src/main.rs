use actix_web::{web, App, Error, HttpResponse, HttpServer};
use scribe::{info, Logger};
use serde::{Deserialize, Serialize};
use wire::types::{AnthropicModel, Message, MessageType, OpenAIModel, API};

#[derive(Serialize, Deserialize)]
struct MemoRequest {
    memo: String,
    genres: String,
}

#[derive(Serialize, Deserialize)]
struct Response {
    emoji: Option<String>,
    name: Option<String>,
    duration: Option<f32>,
    error: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Memo {
    genre: String,
    name: String,
    duration: f32,
    memo: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct WeeklyRequest {
    memos: Vec<Memo>,
}

#[derive(Debug, Serialize, Deserialize)]
struct WeeklyResponse {
    content: Option<String>,
    error: Option<String>,
}

// Thin wrapper over the wire for thread safe mutability
struct WireWrapper {
    inner: std::sync::Mutex<wire::Wire>,
}

async fn extract_memo_data(
    client: &mut wire::Wire,
    memo: &str,
    genre_history: &str,
) -> Result<Response, Box<dyn std::error::Error>> {
    let prompt = format!(
        "Take the following activity memo and return _only_ the schema with the values filled in:\n\n
        '{memo}'\n\n
        Schema: {{\"name\": string, \"emoji\": string, \"duration\": integer}}\n\n
        Take care to use reasonable defaults for, e.g., duration, if needed.\n
        An activity can have multiple emojis associated with it--be thorough in your selection!\n
        Be mindful of the emojis you pick! It should be general and allow for generous overlap with other genres/activities--to this end, here's a recent history of chosen genres (one emoji == one genre):\n\n
        {genre_history}\n\n
        Keep things consistent!\n\n
        As a final note: _Nothing other than JSON should be in your response_--don't wrap it in any markdown formatting or anything"
    );

    let api = API::OpenAI(OpenAIModel::GPT4o);
    let response = client
        .prompt(
            api.clone(),
            &String::new(),
            &vec![Message {
                message_type: MessageType::User,
                content: prompt,
                api,
                system_prompt: String::new(),
            }],
        )
        .await?;

    let parsed = serde_json::from_str(&response.content)?;

    Ok(parsed)
}

async fn handle_memo(
    data: web::Json<MemoRequest>,
    client: web::Data<WireWrapper>,
) -> Result<HttpResponse, Error> {
    if data.memo.is_empty() {
        return Ok(HttpResponse::BadRequest().json(Response {
            emoji: None,
            name: None,
            duration: None,
            error: Some("No activity memo provided".into()),
        }));
    }

    let mut client = client.inner.lock().unwrap();

    match extract_memo_data(&mut client, &data.memo, &data.genres).await {
        Ok(response) => Ok(HttpResponse::Ok().json(response)),
        Err(e) => Ok(HttpResponse::InternalServerError().json(Response {
            emoji: None,
            name: None,
            duration: None,
            error: Some(e.to_string()),
        })),
    }
}

async fn generate_weekly_report(
    client: &mut wire::Wire,
    memos_xml: &str,
) -> Result<WeeklyResponse, Box<dyn std::error::Error>> {
    let prompt = format!(
        "Take the following activity memos and generate a friendly report recounting the user's activities over the past week:\n\n
        '{memos_xml}'\n\n
        Be especially attentive to any latent needs or deficiencies, or abject successes or achievements. Also, be familiar--don't be professionally distant!"
    );

    let api = API::Anthropic(AnthropicModel::Claude35Sonnet);
    let response = client
        .prompt(
            api.clone(),
            &String::new(),
            &vec![Message {
                message_type: MessageType::User,
                content: prompt,
                api,
                system_prompt: String::new(),
            }],
        )
        .await?;

    let response = WeeklyResponse {
        content: Some(response.content),
        error: None,
    };

    Ok(response)
}

async fn handle_weekly(
    data: web::Json<WeeklyRequest>,
    client: web::Data<WireWrapper>,
) -> Result<HttpResponse, Error> {
    let mut client = client.inner.lock().unwrap();
    let data_xml = match serde_json::to_string(&data) {
        Ok(xml) => xml,
        Err(e) => {
            return Ok(HttpResponse::InternalServerError().json(WeeklyResponse {
                error: Some(format!("Failed to serialize request: {}", e)),
                content: None,
            }));
        }
    };

    match generate_weekly_report(&mut client, &data_xml).await {
        Ok(response) => Ok(HttpResponse::Ok().json(response)),
        Err(e) => Ok(HttpResponse::InternalServerError().json(WeeklyResponse {
            content: None,
            error: Some(e.to_string()),
        })),
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    Logger::init(None);

    // No tokenizer file, don't download
    let client = web::Data::new(WireWrapper {
        inner: std::sync::Mutex::new(wire::Wire::new(None, Some(false)).unwrap()),
    });

    let port = "8080";

    info!("Server listening on 127.0.0.1:{}", port);

    HttpServer::new(move || {
        App::new().app_data(client.clone()).service(
            web::scope("/api")
                .route("/memo", web::post().to(handle_memo))
                .route("/weekly", web::post().to(handle_weekly)),
        )
    })
    .bind(format!("127.0.0.1:{}", port))?
    .run()
    .await
}
