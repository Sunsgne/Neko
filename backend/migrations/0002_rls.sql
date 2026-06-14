-- 0002_rls.sql — PostgreSQL 行级安全（多租户隔离, T1.3）
-- 思路：会话通过 `SET app.tenant_id = '<id>'` 设定当前租户；策略仅放行
-- 同租户行。运营端使用 `SET app.tenant_id = ''`（空）表示跨租户（特权）。
-- 应用层（store.PostgresStore）在每次操作前设置该 GUC。

ALTER TABLE devices         ENABLE ROW LEVEL SECURITY;
ALTER TABLE sites           ENABLE ROW LEVEL SECURITY;
ALTER TABLE links           ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_desired  ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_runs     ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts          ENABLE ROW LEVEL SECURITY;

-- current_tenant() 返回当前会话租户；未设置时返回空串（运营/迁移场景）。
CREATE OR REPLACE FUNCTION current_tenant() RETURNS text AS $$
  SELECT coalesce(current_setting('app.tenant_id', true), '');
$$ LANGUAGE sql STABLE;

-- 通用策略：空租户（运营）可见全部；否则仅同租户。
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['devices','sites','links','config_desired','config_runs','alerts']
  LOOP
    EXECUTE format($f$
      CREATE POLICY tenant_isolation ON %I
        USING (current_tenant() = '' OR tenant_id = current_tenant())
        WITH CHECK (current_tenant() = '' OR tenant_id = current_tenant());
    $f$, t);
  END LOOP;
END $$;
