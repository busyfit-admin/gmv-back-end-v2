// Allow company admin/account holders to manage all Employee onboarding services
permit (
    principal == CompanyPortal::Role::"companyAdmin",
    action in
        [CompanyPortal::Action::"getEmployeeUsers",
         CompanyPortal::Action::"postEmployeeUsers",
         CompanyPortal::Action::"deleteEmployeeUsers"],
    resource
)
when { principal.companyId == resource.companyId };