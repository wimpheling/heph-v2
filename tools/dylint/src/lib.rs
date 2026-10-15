#![feature(rustc_private)]

extern crate rustc_errors;
extern crate rustc_hir;

use rustc_errors::DiagDecorator;
use rustc_hir::{Expr, ExprKind};
use rustc_lint::{LateContext, LateLintPass, LintContext};

// Resolving the definition catches aliases and fully qualified std paths.
dylint_linting::declare_late_lint! {
    pub ENV_READS_ONLY_IN_CONFIG,
    Deny,
    "read process environment only in config.rs or config modules"
}

impl<'tcx> LateLintPass<'tcx> for EnvReadsOnlyInConfig {
    fn check_expr(&mut self, cx: &LateContext<'tcx>, expr: &'tcx Expr<'tcx>) {
        let ExprKind::Path(ref qpath) = expr.kind else {
            return;
        };
        let Some(def_id) = cx.qpath_res(qpath, expr.hir_id).opt_def_id() else {
            return;
        };
        let path = cx.tcx.def_path_str(def_id);
        if !matches!(
            path.as_str(),
            "std::env::var" | "std::env::var_os" | "std::env::vars" | "std::env::vars_os"
        ) {
            return;
        }
        let file = cx
            .sess()
            .source_map()
            .span_to_filename(expr.span)
            .prefer_local_unconditionally()
            .to_string()
            .replace('\\', "/");
        if file.ends_with("/config.rs") || file.contains("/config/") || file == "config.rs" {
            return;
        }
        cx.emit_span_lint(ENV_READS_ONLY_IN_CONFIG, expr.span, DiagDecorator(|diag| {
            diag.primary_message("read environment variables in config.rs and pass typed configuration to other modules");
        }));
    }
}

#[test]
fn ui() {
    dylint_testing::ui_test(env!("CARGO_PKG_NAME"), "ui");
}
