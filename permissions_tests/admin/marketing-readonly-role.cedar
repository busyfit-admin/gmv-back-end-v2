// Permit only view Companies actions in Marketing read only Role
permit (
    principal == AdminPortal::Role::"marketingReadOnly",
    action in [NAMESPACE::Action::"GET"],
    resource == NAMESPACE::Endpoint::"/manageCompanies"
);

// Permit only view SubDomain Actions in Marketing read only Role
permit (
    principal == AdminPortal::Role::"marketingReadOnly",
    action in [NAMESPACE::Action::"GET"],
    resource == NAMESPACE::Endpoint::"/manageSubDomains"
);