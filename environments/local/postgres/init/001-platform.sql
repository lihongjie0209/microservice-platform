REVOKE CREATE ON SCHEMA public FROM PUBLIC;

CREATE ROLE identity_service LOGIN PASSWORD 'identity-dev';
CREATE ROLE tenant_service LOGIN PASSWORD 'tenant-dev';
CREATE ROLE authorization_service LOGIN PASSWORD 'authorization-dev';

GRANT CONNECT ON DATABASE platform TO identity_service, tenant_service, authorization_service;

CREATE SCHEMA "identity" AUTHORIZATION identity_service;
CREATE SCHEMA "tenant" AUTHORIZATION tenant_service;
CREATE SCHEMA "authorization" AUTHORIZATION authorization_service;

REVOKE ALL ON SCHEMA "identity", "tenant", "authorization" FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA "identity" TO identity_service;
GRANT USAGE, CREATE ON SCHEMA "tenant" TO tenant_service;
GRANT USAGE, CREATE ON SCHEMA "authorization" TO authorization_service;

ALTER ROLE identity_service IN DATABASE platform SET search_path = "identity";
ALTER ROLE tenant_service IN DATABASE platform SET search_path = "tenant";
ALTER ROLE authorization_service IN DATABASE platform SET search_path = "authorization";
