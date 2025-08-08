
DROP TRIGGER IF EXISTS plugins_notify_delete ON plugins;
DROP TRIGGER IF EXISTS plugins_notify_update ON plugins;
DROP TRIGGER IF EXISTS plugins_notify_insert ON plugins;
DROP FUNCTION IF EXISTS notify_plugin_change();

DROP INDEX IF EXISTS idx_plugin_endpoints_method;
DROP INDEX IF EXISTS idx_plugin_endpoints_plugin_id;
DROP INDEX IF EXISTS idx_plugins_created_at;
DROP INDEX IF EXISTS idx_plugins_type;
DROP INDEX IF EXISTS idx_plugins_status;
DROP INDEX IF EXISTS idx_plugins_name;

DROP TABLE IF EXISTS plugin_endpoints;
DROP TABLE IF EXISTS plugins;
