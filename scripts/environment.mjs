// No hardcoded environment defaults: callers must supply configuration explicitly.
export function requiredEnv(name) {
  const value = process.env[name];
  if (value === undefined || value.trim() === "") {
    throw new Error(name + " must be set to a non-empty value");
  }
  return value;
}

export function booleanEnv(name) {
  const value = requiredEnv(name);
  if (value !== "0" && value !== "1") {
    throw new Error(name + " must be set to 0 or 1");
  }
  return value === "1";
}
