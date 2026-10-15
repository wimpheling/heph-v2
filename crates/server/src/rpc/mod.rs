mod system;

pub fn router() -> connectrpc::Router {
    connectrpc::Router::new().add_service(std::sync::Arc::new(system::SystemService))
}
