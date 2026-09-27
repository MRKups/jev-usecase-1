// Package triage defines the standard ITSM triage questions and evaluation contracts.
package triage

// Question defines an operational question for the Jev model.
type Question struct {
	ID           string            `json:"id"`
	Type         string            `json:"type,omitempty"`
	Text         string            `json:"text"`
	Instructions string            `json:"instructions,omitempty"`
	Options      []string          `json:"options"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

// StandardQuestions returns the canonical questions used to evaluate tickets.
func StandardQuestions() []Question {
	return []Question{
		{
			ID:           "ticket_type",
			Type:         "choice",
			Text:         "Is this ticket an Incident or a Service Request?",
			Instructions: "Determine whether this ticket represents an Incident (unplanned service outage, error, or degradation) or a Service Request (formal request for access, equipment, software, or routine service).",
			Options: []string{
				"Incident",
				"Service Request",
			},
			Criteria: map[string]string{
				"Incident":        "An unplanned interruption, service outage, performance degradation, software crash, hardware malfunction, or system error affecting existing services.",
				"Service Request": "A formal request for new access, software installation, hardware provisioning, password assistance, or administrative change where existing systems operate normally.",
			},
		},
		{
			ID:           "technical_domain",
			Type:         "choice",
			Text:         "What is the primary technical domain of this ticket?",
			Instructions: "Categorize the IT service management ticket into its primary technical domain.",
			Options: []string{
				"Hardware",
				"Software",
				"Network",
				"Access & Identity",
			},
			Criteria: map[string]string{
				"Hardware":          "Physical machines and peripherals: laptops, monitors, docks, keyboards, cables, and printers.",
				"Software":          "Operating systems, desktop tools, web applications, SaaS platforms, and software features.",
				"Network":           "Connectivity: office Wi-Fi, Ethernet, VPN, DNS, firewalls, and internet access.",
				"Access & Identity": "Authentication and authorization: logins, passwords, MFA tokens, permissions, and accounts.",
			},
		},
		{
			ID:           "operational_urgency",
			Type:         "choice",
			Text:         "What is the verified operational urgency level based on technical impact?",
			Instructions: "Assess the operational urgency based on business impact and user blockages.",
			Options: []string{
				"Critical",
				"High",
				"Medium",
				"Low",
			},
			Criteria: map[string]string{
				"Critical": "Complete outage of revenue-critical systems, major security breach, or executive work stoppage with no workaround.",
				"High":     "Severe degradation of key workflows or large department blocked with difficult workaround.",
				"Medium":   "Standard single-user blocker with available workaround or moderate departmental disruption.",
				"Low":      "Informational requests, minor cosmetic bugs, routine access requests, or planned future changes.",
			},
		},
		{
			ID:           "security_incident",
			Type:         "noul",
			Text:         "Does this issue indicate a security breach, unauthorized access, or policy violation?",
			Instructions: "Determine if this ticket represents an active security incident requiring SecOps review.",
			Options: []string{
				"Yes",
				"No",
			},
			Criteria: map[string]string{
				"true":  "Phishing, suspicious login from foreign IP, malware, credential compromise, data exfiltration, lost unencrypted device, or security policy breach.",
				"false": "Routine access request, ordinary hardware or software fault, non-malicious forgotten password, or standard IT troubleshooting.",
			},
		},
		{
			ID:           "target_resolution_group",
			Type:         "choice",
			Text:         "Which support tier or engineering team should resolve this issue?",
			Instructions: "Identify the primary support or engineering group responsible for resolving this issue.",
			Options: []string{
				"Service Desk",
				"Network Operations",
				"Identity & Access Management",
				"Site Reliability Engineering",
				"SecOps",
			},
			Criteria: map[string]string{
				"Service Desk":                 "Frontline support: user equipment, workstation setup, local software issues, and routine fixes.",
				"Network Operations":           "Network infrastructure: Wi-Fi access points, routers, switches, VPN gateways, and firewalls.",
				"Identity & Access Management": "Directory services, SSO configuration, role-based access, and enterprise permissions.",
				"Site Reliability Engineering": "Production infrastructure, cloud servers, backend database issues, or CI/CD deployments.",
				"SecOps":                       "Security incidents, compromised accounts, malware investigations, or security audit alerts.",
			},
		},
		{
			ID:           "blast_radius",
			Type:         "choice",
			Text:         "Does this issue affect an isolated user or indicate a broader systemic outage?",
			Instructions: "Determine the scope and blast radius of the reported incident.",
			Options: []string{
				"Single User",
				"Multiple Users",
				"Company-wide Outage",
			},
			Criteria: map[string]string{
				"Single User":         "Issue is isolated to one individual person or workstation.",
				"Multiple Users":      "Issue affects multiple employees, a project team, or an office area.",
				"Company-wide Outage": "Issue affects all employees across the organization or customer-facing operations.",
			},
		},
	}
}
