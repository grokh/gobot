-- One-time fixes for toril.db's existing data, November 2026. Safe to run
-- more than once, and while the bot is running. Back up first: gobot -bak
--     sqlite3 /srv/bot/toril.db < fix_times_and_stats.sql
--
-- 1. full_stats that came in through import.sql's old Go-style quoting
--    have a backslash and an n (or t) where the identify text had a line
--    break (or tab).
-- 2. Stored times move to UTC with an explicit +00:00, the way the bot
--    writes them now. Times with an offset convert exactly; the oldest rows
--    have no offset and were already UTC. Rows SQLite can't read as a time
--    are left as they are.
BEGIN;

UPDATE items
	SET full_stats = replace(replace(full_stats, '\n', char(10)), '\t', char(9))
	WHERE instr(full_stats, '\') > 0;

UPDATE chars
	SET last_seen = strftime('%Y-%m-%d %H:%M:%f', last_seen) || '+00:00'
	WHERE last_seen NOT LIKE '%+00:00' AND julianday(last_seen) IS NOT NULL;

UPDATE boots
	SET boot_time = strftime('%Y-%m-%d %H:%M:%f', boot_time) || '+00:00'
	WHERE boot_time NOT LIKE '%+00:00' AND julianday(boot_time) IS NOT NULL;

UPDATE loads
	SET report_time = strftime('%Y-%m-%d %H:%M:%f', report_time) || '+00:00'
	WHERE report_time NOT LIKE '%+00:00' AND julianday(report_time) IS NOT NULL;

COMMIT;
