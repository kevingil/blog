
DROP INDEX IF EXISTS idx_task_run_event_step_created;
DROP INDEX IF EXISTS idx_task_run_event_run_created;
DROP INDEX IF EXISTS idx_task_run_step_run_created;
DROP INDEX IF EXISTS idx_task_run_parent_run_id;
DROP INDEX IF EXISTS idx_task_run_user_created;
DROP INDEX IF EXISTS idx_task_run_org_created;
DROP INDEX IF EXISTS idx_task_run_status_created;
DROP INDEX IF EXISTS idx_task_run_task_name_created;
DROP INDEX IF EXISTS idx_task_run_created;

DROP TABLE IF EXISTS task_run_event;
DROP TABLE IF EXISTS task_run_step;
DROP TABLE IF EXISTS task_run;

