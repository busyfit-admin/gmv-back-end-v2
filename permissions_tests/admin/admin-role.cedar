// Permit all Manage Companies actions in Admin Role
permit (
    principal == AdminPortal::Role::"admin",
    action in
        [AdminPortal::Action::"GetCompanyDetails",
         AdminPortal::Action::"PostCompanyDetails",
         AdminPortal::Action::"DeleteCompanyDetails"],
    resource == AdminPortal::Endpoint::"ManageCompanies"
);

// Permit all SubDomain Actions in Admin Role
permit (
    principal == AdminPortal::Role::"admin",
    action in
        [AdminPortal::Action::"GetSubDomains",
         AdminPortal::Action::"PostSubDomains",
         AdminPortal::Action::"DeleteSubDomains"],
    resource == AdminPortal::Endpoint::"ManageSubDomains"
);