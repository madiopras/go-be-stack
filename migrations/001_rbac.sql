-- RBAC: roles
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- RBAC: permissions
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- RBAC: user_roles (many-to-many users <-> roles)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- RBAC: role_permissions (many-to-many roles <-> permissions)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

-- Organizations
CREATE TABLE IF NOT EXISTS organizations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Organization membership + optional role within organization
CREATE TABLE IF NOT EXISTS organization_users (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_organization_users_org_id ON organization_users(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_users_user_id ON organization_users(user_id);

-- Seed default roles
INSERT INTO roles (name, code, description) VALUES
    ('Super Admin', 'super_admin', 'Full system access'),
    ('Admin', 'admin', 'Administrator with most permissions'),
    ('User', 'user', 'Standard user')
ON CONFLICT (code) DO NOTHING;

-- Seed default permissions (resource:action)
INSERT INTO permissions (name, code, resource, action, description) VALUES
    ('List users', 'users:list', 'users', 'list', 'View list of users'),
    ('View user', 'users:read', 'users', 'read', 'View single user'),
    ('Create user', 'users:create', 'users', 'create', 'Create new user'),
    ('Update user', 'users:update', 'users', 'update', 'Update user'),
    ('Delete user', 'users:delete', 'users', 'delete', 'Delete user'),
    ('Manage roles', 'roles:manage', 'roles', 'manage', 'CRUD roles and assign to users'),
    ('Manage permissions', 'permissions:manage', 'permissions', 'manage', 'Manage role permissions'),
    ('Manage organizations', 'organizations:manage', 'organizations', 'manage', 'CRUD organizations and members')
ON CONFLICT (code) DO NOTHING;

-- Assign permissions to roles (super_admin gets all via application or add all here)
-- Admin: users + roles + organizations (no permissions:manage)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'admin' AND p.code IN (
    'users:list', 'users:read', 'users:create', 'users:update', 'users:delete',
    'roles:manage', 'organizations:manage'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- User: read-only users
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'user' AND p.code IN ('users:list', 'users:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Super admin: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.code = 'super_admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
