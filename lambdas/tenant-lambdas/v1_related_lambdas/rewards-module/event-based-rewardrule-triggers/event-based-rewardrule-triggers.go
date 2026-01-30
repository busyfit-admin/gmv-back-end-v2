package main

import (
	"context"
	"log"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

func main() {

}

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
}
