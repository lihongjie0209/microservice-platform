# platform-library

Helm library chart for all platform services. Consumer charts include it as a dependency and render the named `platform.deployment`, `platform.service`, `platform.serviceAccount`, `platform.migrationJob`, `platform.podDisruptionBudget`, `platform.horizontalPodAutoscaler`, `platform.networkPolicy`, `platform.serviceMonitor`, and `platform.externalSecret` templates. Database migration runs as a pre-install/pre-upgrade hook, while the application still retains advisory-lock protection for safe concurrent startup.
