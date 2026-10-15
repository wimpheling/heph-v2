//! Transport-independent application types for the forge and agent runtime.

pub struct SystemInfo {
    pub name: &'static str,
    pub version: &'static str,
}

pub fn system_info() -> SystemInfo {
    SystemInfo {
        name: "heph-v2",
        version: env!("CARGO_PKG_VERSION"),
    }
}
