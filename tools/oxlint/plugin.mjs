import path from "node:path";

function filename(context) {
  return (context.filename ?? context.getFilename()).replaceAll("\\", "/");
}

function appFile(context) {
  return filename(context).includes("/apps/web/src/");
}

function resolveImport(context, source) {
  return source.startsWith(".")
    ? path.resolve(path.dirname(filename(context)), source).replaceAll("\\", "/")
    : source;
}

function imports(check) {
  return {
    ImportDeclaration: (node) => check(node.source),
    ExportNamedDeclaration: (node) => {
      if (node.source) check(node.source);
    },
    ExportAllDeclaration: (node) => check(node.source),
    ImportExpression: (node) => check(node.source),
    CallExpression: (node) => {
      if (node.callee.type === "Identifier" && node.callee.name === "require")
        check(node.arguments[0]);
    },
  };
}

export default {
  meta: { name: "heph" },
  rules: {
    "design-system-boundary": {
      meta: {
        type: "problem",
        schema: [],
        messages: {
          import: "Import UI and styles through @heph/design-system.",
          element: "Use a component from @heph/design-system instead of raw DOM JSX.",
        },
      },
      create(context) {
        if (!appFile(context)) return {};
        return {
          ...imports((node) => {
            const source = node?.value;
            if (typeof source !== "string") return;
            const resolved = resolveImport(context, source);
            if (
              source.startsWith("@astryxdesign/") ||
              source.startsWith("@stylexjs/") ||
              resolved.includes("/packages/design-system/") ||
              (/\.(css|scss|sass|less)(\?|$)/.test(source) &&
                source !== "@heph/design-system/styles.css")
            ) {
              context.report({ node, messageId: "import" });
            }
          }),
          JSXOpeningElement(node) {
            if (node.name.type === "JSXIdentifier" && /^[a-z]/.test(node.name.name))
              context.report({ node, messageId: "element" });
          },
        };
      },
    },
    "rpc-boundary": {
      meta: {
        type: "problem",
        schema: [],
        messages: {
          import: "Keep Connect and protobuf imports in app/rpc; call the application adapter.",
          fetch: "Use the typed RPC adapter instead of direct network calls.",
        },
      },
      create(context) {
        if (!appFile(context) || filename(context).includes("/app/rpc/")) return {};
        const visitors = imports((node) => {
          const source = node?.value;
          if (typeof source !== "string") return;
          const resolved = resolveImport(context, source);
          if (
            /^@(connectrpc|bufbuild)\//.test(source) ||
            source.startsWith("@heph/contracts") ||
            resolved.includes("/packages/contracts/")
          )
            context.report({ node, messageId: "import" });
        });
        return {
          ...visitors,
          CallExpression(node) {
            visitors.CallExpression(node);
            const callee = node.callee;
            if (
              (callee.type === "Identifier" && callee.name === "fetch") ||
              (callee.type === "MemberExpression" &&
                ["window", "globalThis"].includes(callee.object.name) &&
                (callee.property.name === "fetch" || callee.property.value === "fetch"))
            )
              context.report({ node, messageId: "fetch" });
          },
          NewExpression(node) {
            if (["XMLHttpRequest", "WebSocket", "EventSource"].includes(node.callee.name))
              context.report({ node, messageId: "fetch" });
          },
        };
      },
    },
  },
};
