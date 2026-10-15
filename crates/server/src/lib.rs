pub mod config;
mod rpc;

pub fn app() -> axum::Router {
    rpc::router().into_axum_router()
}

#[cfg(test)]
mod tests;
