REVOKE CREATE ON SCHEMA public FROM PUBLIC;

ALTER DATABASE platform SET timezone TO 'Asia/Shanghai';

DO $$
DECLARE
    role_name text;
BEGIN
    FOREACH role_name IN ARRAY ARRAY[
        'identity_service', 'tenant_service', 'authorization_service',
        'audit_service', 'config_service', 'notification_service', 'file_service',
        'scheduler_service', 'application_service', 'dictionary_service',
        'webhook_service', 'workflow_service', 'search_service', 'metering_service',
        'billing_service'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
            EXECUTE format('CREATE ROLE %I LOGIN', role_name);
        END IF;
    END LOOP;
END
$$;

ALTER ROLE identity_service LOGIN PASSWORD 'identity-dev';
ALTER ROLE tenant_service LOGIN PASSWORD 'tenant-dev';
ALTER ROLE authorization_service LOGIN PASSWORD 'authorization-dev';
ALTER ROLE audit_service LOGIN PASSWORD 'audit-dev';
ALTER ROLE config_service LOGIN PASSWORD 'config-dev';
ALTER ROLE notification_service LOGIN PASSWORD 'notification-dev';
ALTER ROLE file_service LOGIN PASSWORD 'file-dev';
ALTER ROLE scheduler_service LOGIN PASSWORD 'scheduler-dev';
ALTER ROLE application_service LOGIN PASSWORD 'application-dev';
ALTER ROLE dictionary_service LOGIN PASSWORD 'dictionary-dev';
ALTER ROLE webhook_service LOGIN PASSWORD 'webhook-dev';
ALTER ROLE workflow_service LOGIN PASSWORD 'workflow-dev';
ALTER ROLE search_service LOGIN PASSWORD 'search-dev';
ALTER ROLE metering_service LOGIN PASSWORD 'metering-dev';
ALTER ROLE billing_service LOGIN PASSWORD 'billing-dev';

GRANT CONNECT ON DATABASE platform TO identity_service, tenant_service, authorization_service, audit_service, config_service, notification_service, file_service, scheduler_service, application_service, dictionary_service, webhook_service, workflow_service, search_service, metering_service, billing_service;

CREATE SCHEMA IF NOT EXISTS "identity" AUTHORIZATION identity_service;
CREATE SCHEMA IF NOT EXISTS "tenant" AUTHORIZATION tenant_service;
CREATE SCHEMA IF NOT EXISTS "authorization" AUTHORIZATION authorization_service;
CREATE SCHEMA IF NOT EXISTS "audit" AUTHORIZATION audit_service;
CREATE SCHEMA IF NOT EXISTS "config" AUTHORIZATION config_service;
CREATE SCHEMA IF NOT EXISTS "notification" AUTHORIZATION notification_service;
CREATE SCHEMA IF NOT EXISTS "file" AUTHORIZATION file_service;
CREATE SCHEMA IF NOT EXISTS "scheduler" AUTHORIZATION scheduler_service;
CREATE SCHEMA IF NOT EXISTS "application" AUTHORIZATION application_service;
CREATE SCHEMA IF NOT EXISTS "dictionary" AUTHORIZATION dictionary_service;
CREATE SCHEMA IF NOT EXISTS "webhook" AUTHORIZATION webhook_service;
CREATE SCHEMA IF NOT EXISTS "workflow" AUTHORIZATION workflow_service;
CREATE SCHEMA IF NOT EXISTS "search" AUTHORIZATION search_service;
CREATE SCHEMA IF NOT EXISTS "metering" AUTHORIZATION metering_service;
CREATE SCHEMA IF NOT EXISTS "billing" AUTHORIZATION billing_service;

ALTER SCHEMA "identity" OWNER TO identity_service;
ALTER SCHEMA "tenant" OWNER TO tenant_service;
ALTER SCHEMA "authorization" OWNER TO authorization_service;
ALTER SCHEMA "audit" OWNER TO audit_service;
ALTER SCHEMA "config" OWNER TO config_service;
ALTER SCHEMA "notification" OWNER TO notification_service;
ALTER SCHEMA "file" OWNER TO file_service;
ALTER SCHEMA "scheduler" OWNER TO scheduler_service;
ALTER SCHEMA "application" OWNER TO application_service;
ALTER SCHEMA "dictionary" OWNER TO dictionary_service;
ALTER SCHEMA "webhook" OWNER TO webhook_service;
ALTER SCHEMA "workflow" OWNER TO workflow_service;
ALTER SCHEMA "search" OWNER TO search_service;
ALTER SCHEMA "metering" OWNER TO metering_service;
ALTER SCHEMA "billing" OWNER TO billing_service;

REVOKE ALL ON SCHEMA "identity", "tenant", "authorization", "audit", "config", "notification", "file", "scheduler", "application", "dictionary", "webhook", "workflow", "search", "metering", "billing" FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA "identity" TO identity_service;
GRANT USAGE, CREATE ON SCHEMA "tenant" TO tenant_service;
GRANT USAGE, CREATE ON SCHEMA "authorization" TO authorization_service;
GRANT USAGE, CREATE ON SCHEMA "audit" TO audit_service;
GRANT USAGE, CREATE ON SCHEMA "config" TO config_service;
GRANT USAGE, CREATE ON SCHEMA "notification" TO notification_service;
GRANT USAGE, CREATE ON SCHEMA "file" TO file_service;
GRANT USAGE, CREATE ON SCHEMA "scheduler" TO scheduler_service;
GRANT USAGE, CREATE ON SCHEMA "application" TO application_service;
GRANT USAGE, CREATE ON SCHEMA "dictionary" TO dictionary_service;
GRANT USAGE, CREATE ON SCHEMA "webhook" TO webhook_service;
GRANT USAGE, CREATE ON SCHEMA "workflow" TO workflow_service;
GRANT USAGE, CREATE ON SCHEMA "search" TO search_service;
GRANT USAGE, CREATE ON SCHEMA "metering" TO metering_service;
GRANT USAGE, CREATE ON SCHEMA "billing" TO billing_service;

ALTER ROLE identity_service IN DATABASE platform SET search_path = "identity";
ALTER ROLE tenant_service IN DATABASE platform SET search_path = "tenant";
ALTER ROLE authorization_service IN DATABASE platform SET search_path = "authorization";
ALTER ROLE audit_service IN DATABASE platform SET search_path = "audit";
ALTER ROLE config_service IN DATABASE platform SET search_path = "config";
ALTER ROLE notification_service IN DATABASE platform SET search_path = "notification";
ALTER ROLE file_service IN DATABASE platform SET search_path = "file";
ALTER ROLE scheduler_service IN DATABASE platform SET search_path = "scheduler";
ALTER ROLE application_service IN DATABASE platform SET search_path = "application";
ALTER ROLE dictionary_service IN DATABASE platform SET search_path = "dictionary";
ALTER ROLE webhook_service IN DATABASE platform SET search_path = "webhook";
ALTER ROLE workflow_service IN DATABASE platform SET search_path = "workflow";
ALTER ROLE search_service IN DATABASE platform SET search_path = "search";
ALTER ROLE metering_service IN DATABASE platform SET search_path = "metering";
ALTER ROLE billing_service IN DATABASE platform SET search_path = "billing";
