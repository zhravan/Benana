CREATE TABLE IF NOT EXISTS plugins (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    version VARCHAR(100) NOT NULL,
    author VARCHAR(255),
    type VARCHAR(100) NOT NULL DEFAULT 'static',
    path VARCHAR(500),
    binary_path VARCHAR(500),
    source_path VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS plugin_endpoints (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    handler VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'REST',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plugins_name ON plugins(name);
CREATE INDEX IF NOT EXISTS idx_plugins_status ON plugins(status);
CREATE INDEX IF NOT EXISTS idx_plugins_type ON plugins(type);
CREATE INDEX IF NOT EXISTS idx_plugins_created_at ON plugins(created_at);
CREATE INDEX IF NOT EXISTS idx_plugin_endpoints_plugin_id ON plugin_endpoints(plugin_id);
CREATE INDEX IF NOT EXISTS idx_plugin_endpoints_method ON plugin_endpoints(method);

CREATE OR REPLACE FUNCTION notify_plugin_change()
RETURNS TRIGGER AS $$
DECLARE
    payload JSON;
BEGIN
    IF TG_OP = 'DELETE' THEN
        payload = json_build_object(
            'operation', TG_OP,
            'table', TG_TABLE_NAME,
            'id', OLD.id,
            'name', OLD.name,
            'old_data', row_to_json(OLD),
            'timestamp', extract(epoch from now())
        );
        PERFORM pg_notify('plugin_changes', payload::text);
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        payload = json_build_object(
            'operation', TG_OP,
            'table', TG_TABLE_NAME,
            'id', NEW.id,
            'name', NEW.name,
            'old_data', row_to_json(OLD),
            'new_data', row_to_json(NEW),
            'timestamp', extract(epoch from now())
        );
        PERFORM pg_notify('plugin_changes', payload::text);
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        payload = json_build_object(
            'operation', TG_OP,
            'table', TG_TABLE_NAME,
            'id', NEW.id,
            'name', NEW.name,
            'new_data', row_to_json(NEW),
            'timestamp', extract(epoch from now())
        );
        PERFORM pg_notify('plugin_changes', payload::text);
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER plugins_notify_insert
    AFTER INSERT ON plugins
    FOR EACH ROW EXECUTE FUNCTION notify_plugin_change();

CREATE TRIGGER plugins_notify_update
    AFTER UPDATE ON plugins
    FOR EACH ROW EXECUTE FUNCTION notify_plugin_change();

CREATE TRIGGER plugins_notify_delete
    AFTER DELETE ON plugins
    FOR EACH ROW EXECUTE FUNCTION notify_plugin_change();
