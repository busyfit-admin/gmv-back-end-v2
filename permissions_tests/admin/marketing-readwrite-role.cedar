// Permits for Companies actions in Marketing read+write only Role
permit (
    principal == AdminPortal::Role::"marketingReadWrite",
    action in [NAMESPACE::Action::"GET", NAMESPACE::Action::"POST"],
    resource == NAMESPACE::Endpoint::"/manageCompanies"
);

// Permits for SubDomain Actions in Marketing read+write only Role
permit (
    principal == AdminPortal::Role::"marketingReadWrite",
    action in [NAMESPACE::Action::"GET", NAMESPACE::Action::"POST"],
    resource == NAMESPACE::Endpoint::"/manageSubDomains"
);