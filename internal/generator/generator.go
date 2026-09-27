// Package generator procedurally builds prompts and drives the LLM ticket generation pipeline.
package generator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/provider"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

// DimensionMatrix contains orthogonal axes for procedural prompt assembly.
type DimensionMatrix struct {
	Departments             []string
	DepartmentRoles         map[string][]string
	Categories              []string
	Systems                 []string
	ProblemClasses          []string
	EmotionalTones          []string
	UrgencyLevels           []string
	CommunicationArchetypes []string
}

// DefaultMatrix returns the standard procedural generation dimensions.
func DefaultMatrix() DimensionMatrix {
	return DimensionMatrix{
		Departments: []string{
			"Engineering",
			"Finance & Accounting",
			"Sales & Commercial",
			"Customer Support & Success",
			"Human Resources & People",
			"Legal & Compliance",
			"Product & Design",
			"Marketing & Communications",
			"Operations & Workplace",
			"Executive Office",
		},
		DepartmentRoles: map[string][]string{
			"Engineering": {
				"Software Engineer",
				"Frontend Developer",
				"Backend Developer",
				"Systems Architect",
				"Site Reliability Engineer",
				"Quality Assurance Engineer",
				"DevOps Engineer",
				"Engineering Manager",
			},
			"Finance & Accounting": {
				"Financial Analyst",
				"Accounts Payable Specialist",
				"Accounts Receivable Specialist",
				"Payroll Administrator",
				"Corporate Controller",
				"Staff Accountant",
				"Billing Coordinator",
				"Tax Analyst",
			},
			"Sales & Commercial": {
				"Account Executive",
				"Sales Representative",
				"Solutions Engineer",
				"Sales Operations Analyst",
				"Account Manager",
				"Business Development Representative",
				"Sales Director",
			},
			"Customer Support & Success": {
				"Support Agent",
				"Customer Success Manager",
				"Client Services Specialist",
				"Technical Support Representative",
				"Implementation Specialist",
				"Customer Onboarding Specialist",
			},
			"Human Resources & People": {
				"Recruiter",
				"HR Generalist",
				"People Operations Coordinator",
				"Benefits Coordinator",
				"Talent Acquisition Specialist",
				"HR Director",
				"Training and Development Specialist",
			},
			"Legal & Compliance": {
				"Corporate Counsel",
				"Paralegal",
				"Contracts Administrator",
				"Compliance Analyst",
				"Legal Assistant",
				"Privacy Officer",
			},
			"Product & Design": {
				"Product Manager",
				"Product Designer",
				"UX Researcher",
				"Technical Writer",
				"Product Operations Manager",
				"UI Designer",
			},
			"Marketing & Communications": {
				"Marketing Specialist",
				"Content Writer",
				"Digital Marketing Manager",
				"Communications Coordinator",
				"Brand Designer",
				"Social Media Coordinator",
				"Events Coordinator",
			},
			"Operations & Workplace": {
				"Operations Coordinator",
				"Facilities Specialist",
				"Supply Chain Analyst",
				"Office Manager",
				"Logistics Coordinator",
				"Procurement Specialist",
			},
			"Executive Office": {
				"Chief Executive Officer",
				"Chief Operating Officer",
				"Chief Financial Officer",
				"Executive Assistant",
				"Chief Technology Officer",
				"Chief Legal Officer",
			},
		},
		Categories: []string{
			"Hardware",
			"Software",
			"Network",
			"Access & Identity",
		},
		Systems: []string{
			"Internal CRM",
			"Telephony System",
			"Chat App",
			"Video Conference System",
			"Single Sign-On Portal",
			"Virtual Private Network Client",
			"Enterprise Resource Planning System",
			"Corporate Email Client",
			"Document Management System",
			"Payroll Portal",
			"Code Repository Server",
			"Continuous Integration Pipeline",
			"Cloud Database Cluster",
			"Employee Intranet",
			"Helpdesk Ticketing System",
			"Network File Share",
			"Office Badge Scanner",
			"Network Printer",
			"Laptop Workstation",
			"Desktop Workstation",
			"Conference Room Display",
			"Expense Reporting Portal",
			"Applicant Tracking System",
			"Endpoint Antivirus Agent",
			"Customer Data Platform",
			"Inventory Management System",
			"Business Intelligence Dashboard",
			"Contract Management Platform",
			"Learning Management System",
			"Wireless Access Point",
		},
		ProblemClasses: []string{
			"Authentication Failure",
			"Access Permission Denied",
			"Connection Timeout",
			"Network Latency Degradation",
			"Application Crash",
			"Data Synchronization Error",
			"Hardware Malfunction",
			"Display Output Failure",
			"Peripheral Connectivity Loss",
			"Software License Expiration",
			"File Corruption",
			"Service Unavailability",
			"Session Disconnection",
			"Performance Degradation",
			"Unexpected System Reboot",
			"Security Alert Warning",
			"Print Spooler Error",
			"Input Device Unresponsive",
			"Storage Quota Exceeded",
			"Audio Video Feed Glitch",
			"Account Lockout",
			"Software Update Failure",
			"Configuration Error",
			"Memory Leak",
			"Software Installation Request",
			"Hardware Equipment Provisioning",
			"System Access Permission Grant",
			"Shared Folder Access Request",
			"Software License Allocation",
			"VPN Profile Setup Request",
			"Peripheral Upgrade Request",
			"Account Provisioning Request",
			"Password Reset Assistance",
			"MFA Token Reset Request",
			"Hardware Replacement Request",
			"Distribution List Membership Request",
		},
		EmotionalTones: []string{
			"Calm and structured",
			"Frustrated",
			"Urgent",
			"Confused",
			"Polite",
			"Concise",
		},
		UrgencyLevels: []string{
			"Low",
			"Medium",
			"High",
			"Critical",
		},
		CommunicationArchetypes: []string{
			"Clear and Structured Professional (concise, balanced, provides exact symptoms, steps, and business impact)",
			"Non-Native English Speaker (limited vocabulary and syntax, phonetic spelling, direct translated idioms)",
			"Terse Mobile Reporter (sent from smartphone, 1-2 fragmented sentences, abbreviations, no capitalization or punctuation)",
			"Minimalist One-Liner (merely states \"not working\" or \"system broken, please fix\" with almost zero context)",
			"Rambling Narrative (long conversational story recounting personal morning routine and meetings before mentioning the bug)",
			"ESL Fluent with Formal Phrasing (advanced vocabulary and technical competence, but uses distinctly formal or archaic phrasing)",
			"Overly Formal Bureaucratic Memo (reads like a legal or executive memo with formal salutations, policy references, and sign-offs)",
			"Casual Chat Style (informal Slack-like conversational style, colloquial phrases, and friendly banter)",
			"Senior Engineer Stack Dump (deeply technical, includes exact error codes, stack traces, reproduction steps, and timestamps)",
			"Vague and Non-Technical (unfamiliar with technical terms, describes visual or physical symptoms vaguely)",
			"Hardware Confuser (confuses monitor with PC tower, router with modem, local drive with cloud)",
			"Raw Terminal Excerpt (pastes raw logs, CLI error output, or JSON payload with minimal human explanation)",
			"Jargon-Heavy Buzzword User (misuses corporate and IT buzzwords to sound technical without conveying actionable details)",
			"New Hire Onboarding (confused about whether an issue is a bug or a missing onboarding permission)",
			"Panicked and Catastrophizing (alarmed, multiple exclamation marks, assumes catastrophic data loss, pleading for immediate help)",
			"Angry and Demanding (combative tone, blames IT for missed personal deadlines, demands immediate escalation to leadership)",
			"Apologetic and Hesitant (timid and excessively polite, worries they broke the computer, apologizes repeatedly)",
			"Self-Troubleshooter (attempted random registry, driver, or setting changes and made the problem worse)",
			"Executive Shortcut (rushed C-level or VP tone, assumes IT knows their setup, demanding immediate white-glove treatment)",
			"All-Caps Urgency (typed entirely in uppercase with repeated question marks, expressing loud urgency)",
			"Frustrated Chronic Reporter (complains that this exact issue happened multiple times this month, expressing system fatigue)",
			"Second-Hand Proxy Reporter (reporting on behalf of a coworker or customer, missing firsthand reproduction details)",
			"Process Skeptic (complains about having to fill out an IT ticket before briefly describing the problem)",
			"Passive-Aggressive User (makes subtle snide remarks about IT responsiveness and lost productivity while explaining the issue)",
		},
	}
}

func sample(items []string) string {
	if len(items) == 0 {
		return ""
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}

// BuildPrompt procedurally combines dimensions into an instruction prompt.
func (d DimensionMatrix) BuildPrompt() string {
	dept := sample(d.Departments)
	jobTitle := ""
	if roles, ok := d.DepartmentRoles[dept]; ok && len(roles) > 0 {
		jobTitle = sample(roles)
	}
	cat := sample(d.Categories)
	system := sample(d.Systems)
	problem := sample(d.ProblemClasses)
	tone := sample(d.EmotionalTones)
	urgency := sample(d.UrgencyLevels)
	archetype := sample(d.CommunicationArchetypes)

	return fmt.Sprintf(`Generate a realistic IT service management (ITSM) support ticket based on the following scenario parameters:
- Department: %s
- Reporter Role: %s
- Category: %s
- Affected System: %s
- Problem Class: %s
- Reported Urgency: %s
- Reporter Emotional Tone: %s
- Communication Archetype: %s

Guidelines:
1. Communication Archetype:
   - The ticket summary and description MUST authentically embody the assigned Communication Archetype.
   - Tailor the employee's language proficiency, vocabulary, grammar, punctuation, sentence length, and technical detail to this archetype. For example, non-native speakers may use translated idioms or simplified grammar; technical engineers provide exact error logs or stack dumps; mobile reporters write terse fragments; panicked users write with high alarm.
   - Do not recite or echo the parameter labels verbatim. Write natural, firsthand employee narrative describing the symptom, workflow disruption, and business impact.

2. Fictitious Reporter Identity:
   - Provide reporter_email as a fictitious Latin or ancient-inspired username in lowercase dot format (first.last).
   - Generate diverse, original first and last names across different requests. Do not default to the same recurring names.
   - Do not use famous historical figures or real living individuals.
   - Do not include an '@' symbol or domain name.

3. Incident or Service Request Nature:
   - Tickets can represent either an Incident (an unplanned disruption, error, performance failure, or system fault) or a Service Request (a request for new access, software installation, hardware provisioning, or administrative support).
   - In realistic IT environments, users often label service requests as urgent incidents or describe incidents without technical labels. Ensure authentic user perspective and wording without quoting formal classification terminology.

Output JSON matching this exact structure:
{
  "reporter_email": "first.last",
  "department": "%s",
  "category": "%s",
  "reported_urgency": "Low|Medium|High|Critical",
  "summary": "Issue title reflecting the reporter's communication style",
  "description": "Firsthand narrative from the employee describing the issue and impact according to their communication archetype.",
  "affected_system": "%s"
}`, dept, jobTitle, cat, system, problem, urgency, tone, archetype, dept, cat, system)
}

// Generator manages the creation of synthetic ITSM tickets.
type Generator struct {
	provider provider.LLMProvider
	matrix   DimensionMatrix
}

// NewGenerator creates a new ticket generator with the provided LLM backend.
func NewGenerator(p provider.LLMProvider, matrix DimensionMatrix) *Generator {
	return &Generator{
		provider: p,
		matrix:   matrix,
	}
}

// GenerateOne produces a single ticket using procedural prompt sampling.
func (g *Generator) GenerateOne(ctx context.Context) (*ticket.Ticket, error) {
	prompt := g.matrix.BuildPrompt()
	t, err := g.provider.GenerateTicket(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("generate ticket: %w", err)
	}
	return t, nil
}

// GenerateBatch creates count tickets.
func (g *Generator) GenerateBatch(ctx context.Context, count int) ([]*ticket.Ticket, error) {
	tickets := make([]*ticket.Ticket, 0, count)
	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return tickets, ctx.Err()
		default:
		}
		t, err := g.GenerateOne(ctx)
		if err != nil {
			return tickets, fmt.Errorf("batch generation item %d: %w", i+1, err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// Stream emits tickets continuously at the given interval until ctx is cancelled.
func (g *Generator) Stream(ctx context.Context, interval time.Duration) <-chan *ticket.Ticket {
	ch := make(chan *ticket.Ticket)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t, err := g.GenerateOne(ctx)
				if err != nil {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case ch <- t:
				}
			}
		}
	}()
	return ch
}
