-- The built-in copilot is gone. Drop the tables and column it left behind.
-- chat_message references article, so drop that child before the other tables.
DROP TABLE IF EXISTS chat_message;
DROP TABLE IF EXISTS agent_skill;
DROP TABLE IF EXISTS mcp_connector;
ALTER TABLE article DROP COLUMN IF EXISTS session_memory;
