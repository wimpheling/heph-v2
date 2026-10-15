use std::{env, net::SocketAddr};

pub struct Config {
    pub listen_addr: SocketAddr,
}

impl Config {
    pub fn from_env() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let listen_addr = match env::var("HEPH_LISTEN_ADDR") {
            Ok(value) => value.parse()?,
            Err(env::VarError::NotPresent) => "127.0.0.1:8080".parse()?,
            Err(error) => return Err(error.into()),
        };
        Ok(Self { listen_addr })
    }
}
