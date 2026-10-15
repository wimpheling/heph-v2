use std::env::var as read_env;

fn main() {
    let _ = read_env("HEPH_LISTEN_ADDR");
    let _ = std::env::var_os("HEPH_LISTEN_ADDR");
    let _ = std::env::vars();
    let _ = std::env::vars_os();
}
