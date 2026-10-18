CREATE TABLE IF NOT EXISTS scopes (
 kind text NOT NULL CHECK (kind IN ('platform','personal','organization','project','resource','git_ref')),
 id text NOT NULL CHECK (length(id) BETWEEN 1 AND 128),
 parent_kind text,
 parent_id text,
 owner_source text,
 owner_subject text,
 resource_kind text NOT NULL DEFAULT '',
 ref_name text NOT NULL DEFAULT '',
 PRIMARY KEY (kind,id),
 FOREIGN KEY (parent_kind,parent_id) REFERENCES scopes(kind,id),
 CHECK ((parent_kind IS NULL) = (parent_id IS NULL)),
 CHECK ((owner_source IS NULL) = (owner_subject IS NULL)),
 CHECK ((kind = 'platform' AND id = 'platform' AND parent_kind IS NULL) OR (kind <> 'platform' AND parent_kind IS NOT NULL)),
 CHECK ((kind = 'personal' AND owner_source IS NOT NULL) OR (kind <> 'personal' AND owner_source IS NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS personal_owner ON scopes(owner_source,owner_subject) WHERE kind='personal';
CREATE UNIQUE INDEX IF NOT EXISTS git_ref_target ON scopes(parent_kind,parent_id,ref_name) WHERE kind='git_ref';
CREATE TABLE IF NOT EXISTS grants (
 id text PRIMARY KEY,
 grantee_source text NOT NULL,
 grantee_subject text NOT NULL,
 action text NOT NULL,
 scope_kind text NOT NULL,
 scope_id text NOT NULL,
 granted_by_source text NOT NULL,
 granted_by_subject text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 revoked_by_source text,
 revoked_by_subject text,
 revoked_at timestamptz,
 FOREIGN KEY(scope_kind,scope_id) REFERENCES scopes(kind,id),
 CHECK ((revoked_at IS NULL) = (revoked_by_source IS NULL)),
 CHECK ((revoked_at IS NULL) = (revoked_by_subject IS NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS active_assignment ON grants(grantee_source,grantee_subject,action,scope_kind,scope_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS active_actor_grants ON grants(grantee_source,grantee_subject) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS grants_scope ON grants(scope_kind,scope_id,id);
CREATE TABLE IF NOT EXISTS bootstrap_state (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 actor_source text NOT NULL,
 actor_subject text NOT NULL
);
