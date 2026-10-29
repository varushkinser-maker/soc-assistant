package detection

import (
	"regexp"
	"soc-assistant/internal/models"
	"strings"
)

type Rule struct {
	RuleName string
	Severity string
	Match    func(*models.Extevent) bool
}

var patern = regexp.MustCompile(`"(from_port|to_port)"\s*:\s*"(22|3389)"`)
var Rules = []Rule{
	{
		RuleName: "Создание IAM-ключа",
		Severity: "Critical",
		Match: func(e *models.Extevent) bool {
			return e.TypeName() == "yandex.cloud.iam.v1.AccessKeyService.Create" || e.TypeName() == "yandex.cloud.iam.v1.ServiceAccountService.CreateKey"
		},
	},
	{
		RuleName: "Удаление или изменение ключа KMS",
		Severity: "Critical",
		Match: func(e *models.Extevent) bool {
			return strings.HasPrefix(e.TypeName(), "yandex.cloud.kms.v1.SymmetricKeyService.") && (strings.HasSuffix(e.TypeName(), "Delete") || strings.HasSuffix(e.TypeName(), "Update"))
		},
	},
	{
		RuleName: "Security Group открыта наружу",
		Severity: "High",
		Match: func(e *models.Extevent) bool {
			return strings.HasPrefix(e.TypeName(), "yandex.cloud.vpc.v1.SecurityGroupService.") &&
				(strings.HasSuffix(e.TypeName(), ".Update") || strings.HasSuffix(e.TypeName(), ".AddRule")) &&
				strings.Contains(e.Raw, "0.0.0.0/0") && patern.MatchString(e.Raw)

		},
	},
}

func Match(e *models.Extevent) *Rule {
	for i := range Rules {
		if Rules[i].Match(e) {
			return &Rules[i]
		}
	}
	return nil
}
