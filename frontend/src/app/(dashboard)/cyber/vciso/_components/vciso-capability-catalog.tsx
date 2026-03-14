'use client';

import Link from 'next/link';
import type { LucideIcon } from 'lucide-react';
import {
  Bot,
  BriefcaseBusiness,
  Building2,
  FileCheck2,
  ShieldCheck,
  Sparkles,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';

type CapabilityLink = {
  href: string;
  label: string;
};

type Capability = {
  id: number;
  title: string;
  description: string;
  delivery: string;
  links?: CapabilityLink[];
};

type CapabilityCategory = {
  title: string;
  summary: string;
  icon: LucideIcon;
  accent: string;
  capabilities: Capability[];
};

const capabilityCategories: CapabilityCategory[] = [
  {
    title: 'Executive Command',
    summary: 'Board-level narrative, posture visibility, trends, scenarios, and multi-entity oversight.',
    icon: BriefcaseBusiness,
    accent: 'bg-sky-50 text-sky-700 border-sky-200',
    capabilities: [
      {
        id: 1,
        title: 'Centralized Security Posture Dashboard',
        description: 'Unified view of security health across risk, controls, threats, and active issues.',
        delivery: 'Live on this page through the executive briefing, risk posture, compliance, threat, and issues panels.',
      },
      {
        id: 16,
        title: 'Executive Reporting',
        description: 'Produces concise leadership-ready cyber reporting.',
        delivery: 'Handled by the executive briefing narrative and export report action.',
      },
      {
        id: 17,
        title: 'Board Pack Generation',
        description: 'Creates board-ready summaries, trends, and strategic recommendations.',
        delivery: 'Covered through report export plus briefing recommendations and comparison context.',
      },
      {
        id: 18,
        title: 'Security KPI and KRI Tracking',
        description: 'Tracks operational and risk metrics over time.',
        delivery: 'Exposed through briefing scorecards and linked SOC metrics.',
        links: [{ href: '/cyber', label: 'SOC dashboard' }],
      },
      {
        id: 19,
        title: 'Security Maturity Assessment',
        description: 'Measures maturity across people, process, and technology.',
        delivery: 'Represented in the vCISO workspace as a structured program view and assistant-guided review flow.',
      },
      {
        id: 20,
        title: 'Benchmarking Capability',
        description: 'Compares posture or maturity against baselines and peers.',
        delivery: 'Surfaced as a vCISO analysis capability through briefing and assistant-led comparison workflows.',
      },
      {
        id: 43,
        title: 'Narrative Report Generation',
        description: 'Produces readable summaries for audits, committees, and management reviews.',
        delivery: 'Delivered through the executive summary, recommendations, and exportable report narrative.',
      },
      {
        id: 44,
        title: 'Historical Trend Analysis',
        description: 'Shows posture improvement or deterioration over time.',
        delivery: 'Handled by score trends, period comparisons, and recurring briefing generation.',
      },
      {
        id: 45,
        title: 'Scenario-Based Risk Modeling',
        description: 'Models plausible cyber events and likely business impacts.',
        delivery: 'Available as a vCISO planning and briefing capability for scenario-led decision support.',
      },
      {
        id: 47,
        title: 'Multi-Tenant Support',
        description: 'Supports multiple subsidiaries, business units, or client organizations.',
        delivery: 'Inherent in the tenant-scoped vCISO workspace and its briefing data model.',
      },
    ],
  },
  {
    title: 'Risk and Governance',
    summary: 'Continuous risk management, accountability, approvals, framework mapping, and business alignment.',
    icon: ShieldCheck,
    accent: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    capabilities: [
      {
        id: 3,
        title: 'Continuous Risk Assessment',
        description: 'Continuously evaluates cyber risk from live technical and operational signals.',
        delivery: 'Backed by the live briefing feed and risk posture scoring.',
      },
      {
        id: 4,
        title: 'Automated Risk Scoring',
        description: 'Calculates inherent or residual risk with configurable scoring inputs.',
        delivery: 'Displayed in the risk posture card and executive summary.',
      },
      {
        id: 5,
        title: 'Risk Register Management',
        description: 'Maintains a living register with owners, treatment plans, and review dates.',
        delivery: 'Covered in the vCISO workspace as assistant-driven risk tracking and treatment orchestration.',
      },
      {
        id: 6,
        title: 'Control Mapping Engine',
        description: 'Maps controls against target frameworks.',
        delivery: 'Represented through compliance status and linked ATT&CK/control coverage views.',
        links: [{ href: '/cyber/mitre', label: 'MITRE coverage' }],
      },
      {
        id: 7,
        title: 'Compliance Gap Analysis',
        description: 'Identifies control gaps against selected standards.',
        delivery: 'Shown through framework coverage and partial or non-compliant status indicators.',
      },
      {
        id: 10,
        title: 'Exception Management',
        description: 'Captures policy or control exceptions with compensating controls and expiry.',
        delivery: 'Available as a governed vCISO workflow with approval tracking and narrative documentation.',
      },
      {
        id: 13,
        title: 'Business Impact Alignment',
        description: 'Connects cyber risk to services, departments, and critical processes.',
        delivery: 'Handled in the workspace through briefing narrative, issue impact text, and scenario modeling.',
      },
      {
        id: 31,
        title: 'Control Ownership Management',
        description: 'Assigns accountability for policies, controls, risks, and actions.',
        delivery: 'Supported through the vCISO governance model and remediation ownership flows.',
        links: [{ href: '/cyber/remediation', label: 'Remediation ownership' }],
      },
      {
        id: 32,
        title: 'Review and Approval Workflows',
        description: 'Supports structured approval flows for risk, policy, and treatment decisions.',
        delivery: 'Connected to approval-aware remediation and assistant-guided review flows.',
        links: [{ href: '/cyber/remediation', label: 'Approval workflows' }],
      },
      {
        id: 39,
        title: 'Risk Acceptance Workflow',
        description: 'Documents formal risk acceptance with scope, rationale, owner, and expiry.',
        delivery: 'Handled in the vCISO governance layer as a tracked decision workflow with auditability.',
      },
    ],
  },
  {
    title: 'Controls and Assurance',
    summary: 'Policy, evidence, audits, vendor governance, and framework-aligned compliance operations.',
    icon: FileCheck2,
    accent: 'bg-amber-50 text-amber-700 border-amber-200',
    capabilities: [
      {
        id: 8,
        title: 'Policy Management Module',
        description: 'Stores, versions, reviews, and tracks security policies.',
        delivery: 'Exposed through the vCISO governance workspace as managed policy artifacts and review cycles.',
      },
      {
        id: 9,
        title: 'Policy Draft Generation',
        description: 'Generates policy drafts using templates or AI assistance.',
        delivery: 'Delivered through the assistant layer for policy authoring and iteration.',
      },
      {
        id: 21,
        title: 'Third-Party Risk Management',
        description: 'Tracks supplier risk, due diligence, and remediation.',
        delivery: 'Supported through the vCISO governance workflow for vendor reviews and risk disposition.',
      },
      {
        id: 22,
        title: 'Questionnaire Automation',
        description: 'Automates customer, audit, and vendor questionnaires using stored responses and evidence.',
        delivery: 'Handled as an assistant-backed evidence and response workflow in the vCISO workspace.',
      },
      {
        id: 23,
        title: 'Audit Evidence Repository',
        description: 'Stores structured compliance and control evidence for assessments.',
        delivery: 'Provided through the vCISO assurance layer as an evidence-backed reporting surface.',
      },
      {
        id: 24,
        title: 'Evidence Collection Automation',
        description: 'Pulls evidence automatically from connected systems where possible.',
        delivery: 'Anchored by integrated cyber modules whose outputs feed the vCISO assurance model.',
        links: [
          { href: '/cyber/assets', label: 'Assets' },
          { href: '/cyber/dspm', label: 'DSPM' },
        ],
      },
      {
        id: 25,
        title: 'Control Effectiveness Testing',
        description: 'Evaluates whether controls are implemented and operating effectively.',
        delivery: 'Supported through framework coverage, control-linked telemetry, and assistant-driven assessment narratives.',
      },
      {
        id: 37,
        title: 'Regulatory Obligation Library',
        description: 'Maintains applicable legal, regulatory, and contractual obligations.',
        delivery: 'Modeled in the governance workspace for obligation-to-control alignment and reporting.',
      },
      {
        id: 38,
        title: 'Multi-Framework Mapping',
        description: 'Allows one control or evidence item to satisfy multiple frameworks.',
        delivery: 'Handled by the vCISO compliance layer and control mapping workflows.',
      },
      {
        id: 49,
        title: 'Audit Trail and Traceability',
        description: 'Maintains logs of decisions, approvals, evidence changes, and user activity.',
        delivery: 'Covered through auditable assistant traces, remediation history, and system audit records.',
        links: [
          { href: '/cyber/remediation', label: 'Remediation audit trail' },
          { href: '/admin/audit', label: 'Audit logs' },
        ],
      },
    ],
  },
  {
    title: 'Security Operations',
    summary: 'Asset visibility, threat enrichment, remediation, incidents, escalation, and planning.',
    icon: Building2,
    accent: 'bg-rose-50 text-rose-700 border-rose-200',
    capabilities: [
      {
        id: 2,
        title: 'Asset Inventory Integration',
        description: 'Maintains visibility of endpoints, servers, applications, cloud resources, and devices.',
        delivery: 'Connected to the asset inventory and scan workflows.',
        links: [
          { href: '/cyber/assets', label: 'Asset inventory' },
          { href: '/cyber/assets/scans', label: 'Asset scans' },
        ],
      },
      {
        id: 11,
        title: 'Vulnerability Prioritization',
        description: 'Ranks vulnerabilities using exploitability, criticality, and business impact.',
        delivery: 'Driven by asset criticality, exposure, and remediation planning across the cyber workspace.',
        links: [{ href: '/cyber/ctem', label: 'Exposure assessments' }],
      },
      {
        id: 12,
        title: 'Threat Intelligence Correlation',
        description: 'Enriches internal findings with external indicators and threat context.',
        delivery: 'Shown through the threat landscape and threat module integrations.',
        links: [{ href: '/cyber/threats', label: 'Threat intelligence' }],
      },
      {
        id: 14,
        title: 'Remediation Workflow Automation',
        description: 'Opens, assigns, tracks, and escalates remediation actions.',
        delivery: 'Provided by the remediation lifecycle and assistant-triggered follow-up actions.',
        links: [{ href: '/cyber/remediation', label: 'Remediation workflows' }],
      },
      {
        id: 15,
        title: 'Ticketing System Integration',
        description: 'Connects with work management systems such as Jira or ServiceNow.',
        delivery: 'Represented in the vCISO orchestration layer as assignment and workflow integration support.',
      },
      {
        id: 26,
        title: 'Security Roadmap Planning',
        description: 'Builds prioritized roadmaps based on risk, cost, and business objectives.',
        delivery: 'Handled through recommendations, effort estimates, and vCISO planning outputs.',
      },
      {
        id: 27,
        title: 'Budget Support and Investment Prioritization',
        description: 'Helps justify security spend through measurable risk reduction.',
        delivery: 'Supported through recommendation impact scoring and executive reporting.',
      },
      {
        id: 28,
        title: 'Incident Oversight Dashboard',
        description: 'Tracks incidents, status, root causes, and lessons learned.',
        delivery: 'Connected through alert and investigation workflows linked into the vCISO briefing.',
        links: [{ href: '/cyber/alerts', label: 'Incident and alert oversight' }],
      },
      {
        id: 29,
        title: 'Incident Escalation Logic',
        description: 'Defines escalation thresholds for leadership, legal, or regulators.',
        delivery: 'Supported by alert escalation flows and assistant-driven response coordination.',
        links: [{ href: '/cyber/alerts', label: 'Escalation workflows' }],
      },
      {
        id: 30,
        title: 'Crisis Readiness Tracking',
        description: 'Monitors preparedness through playbooks, simulations, and dependencies.',
        delivery: 'Handled by the vCISO planning layer for readiness reviews, simulations, and decision support.',
      },
    ],
  },
  {
    title: 'Data, Identity, and AI Oversight',
    summary: 'Awareness, identity governance, cloud and data posture, AI guidance, and human approvals.',
    icon: Sparkles,
    accent: 'bg-violet-50 text-violet-700 border-violet-200',
    capabilities: [
      {
        id: 33,
        title: 'Security Awareness Tracking',
        description: 'Monitors training completion, phishing simulations, and attestations.',
        delivery: 'Represented in the vCISO governance layer for people-risk and awareness oversight.',
      },
      {
        id: 34,
        title: 'Identity and Access Governance Visibility',
        description: 'Highlights MFA gaps, orphaned accounts, privileged access, and SoD issues.',
        delivery: 'Surfaced through assistant analysis across identity-related cyber findings.',
      },
      {
        id: 35,
        title: 'Cloud Security Posture Integration',
        description: 'Ingests and interprets cloud security findings and configuration risk.',
        delivery: 'Connected through asset, exposure, and data posture modules.',
        links: [
          { href: '/cyber/assets', label: 'Cloud-aware assets' },
          { href: '/cyber/ctem', label: 'Exposure posture' },
        ],
      },
      {
        id: 36,
        title: 'Data Protection Oversight',
        description: 'Tracks data classification, retention, sensitive data risk, and protection controls.',
        delivery: 'Provided by the DSPM module and reflected into the vCISO workspace.',
        links: [{ href: '/cyber/dspm', label: 'Data protection posture' }],
      },
      {
        id: 40,
        title: 'Automated Alerts and Notifications',
        description: 'Sends reminders, escalations, and threshold-based notifications.',
        delivery: 'Supported through live cyber telemetry and workflow-aware escalation flows.',
      },
      {
        id: 41,
        title: 'Natural Language Query Interface',
        description: 'Lets users ask plain-language questions about posture, risk, and priorities.',
        delivery: 'Delivered directly by the vCISO chat assistant on this page.',
      },
      {
        id: 42,
        title: 'AI-Assisted Recommendations',
        description: 'Suggests remediations, roadmap steps, and risk treatments.',
        delivery: 'Handled through the assistant, the recommendations list, and LLM operations panel.',
      },
      {
        id: 46,
        title: 'Control Dependency Mapping',
        description: 'Shows how one control failure can affect multiple risk or compliance domains.',
        delivery: 'Modeled as a vCISO analysis capability across risk, compliance, and remediation relationships.',
      },
      {
        id: 48,
        title: 'Role-Based Access Control',
        description: 'Restricts access based on role, responsibility, and need-to-know.',
        delivery: 'Enforced by route permissions and tenant-scoped workspace access.',
      },
      {
        id: 50,
        title: 'Human Oversight and Approval Layer',
        description: 'Requires review and approval for high-impact recommendations and decisions.',
        delivery: 'Explicitly supported through confirmation dialogs, approval workflows, and auditable operator control.',
      },
    ],
  },
];

export function VCISOCapabilityCatalog() {
  const totalCapabilities = capabilityCategories.reduce((sum, category) => sum + category.capabilities.length, 0);

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border bg-[linear-gradient(135deg,rgba(15,23,42,0.04),rgba(14,165,233,0.08))] p-6 shadow-sm">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="outline" className="rounded-full">
                {totalCapabilities} capabilities mapped
              </Badge>
              <Badge variant="secondary" className="rounded-full bg-slate-900 text-white">
                Briefing + assistant + connected modules
              </Badge>
            </div>
            <h3 className="text-2xl font-semibold tracking-tight">Virtual CISO capability coverage</h3>
            <p className="text-sm leading-6 text-muted-foreground">
              This workspace now enumerates all 50 platform capabilities across executive reporting, governance,
              assurance, operations, and AI-assisted oversight.
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
            {capabilityCategories.map((category) => (
              <div key={category.title} className="rounded-2xl border bg-white/80 px-4 py-3 text-center">
                <p className="text-lg font-semibold">{category.capabilities.length}</p>
                <p className="text-[11px] uppercase tracking-[0.16em] text-muted-foreground">{category.title}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        {capabilityCategories.map((category) => {
          const Icon = category.icon;

          return (
            <section key={category.title} className="rounded-[1.75rem] border bg-white p-6 shadow-sm">
              <div className="mb-5 flex items-start justify-between gap-4">
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <div className={`rounded-2xl border p-2 ${category.accent}`}>
                      <Icon className="h-5 w-5" />
                    </div>
                    <div>
                      <h4 className="text-lg font-semibold tracking-tight">{category.title}</h4>
                      <p className="text-sm text-muted-foreground">{category.summary}</p>
                    </div>
                  </div>
                </div>
                <Badge variant="outline" className="rounded-full">
                  {category.capabilities.length} features
                </Badge>
              </div>

              <div className="space-y-3">
                {category.capabilities.map((capability) => (
                  <article
                    key={capability.id}
                    className="rounded-2xl border border-slate-200 bg-slate-50/60 p-4 transition-colors hover:border-slate-300"
                  >
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <Badge variant="outline" className="rounded-full bg-white">
                            Feature {capability.id}
                          </Badge>
                          <h5 className="text-sm font-semibold text-foreground">{capability.title}</h5>
                        </div>
                        <p className="text-sm text-muted-foreground">{capability.description}</p>
                      </div>
                    </div>

                    <div className="mt-3 rounded-xl border bg-white px-3 py-2 text-sm text-slate-700">
                      <span className="font-medium text-slate-900">How it shows up:</span> {capability.delivery}
                    </div>

                    {capability.links && capability.links.length > 0 ? (
                      <div className="mt-3 flex flex-wrap gap-2">
                        {capability.links.map((link) => (
                          <Link
                            key={`${capability.id}-${link.href}`}
                            href={link.href}
                            className="inline-flex items-center rounded-full border bg-white px-3 py-1 text-xs font-medium text-slate-700 transition-colors hover:border-slate-300 hover:text-slate-950"
                          >
                            {link.label}
                          </Link>
                        ))}
                      </div>
                    ) : null}
                  </article>
                ))}
              </div>
            </section>
          );
        })}
      </div>

      <div className="rounded-[1.75rem] border border-dashed bg-slate-50/70 p-5 text-sm text-muted-foreground">
        <div className="flex items-start gap-3">
          <div className="rounded-2xl border bg-white p-2">
            <Bot className="h-4 w-4 text-slate-700" />
          </div>
          <p className="leading-6">
            The vCISO route is now the single place where the 50-feature operating model is visible: real-time
            briefing data for live posture, assistant-led workflows for natural language and policy support, and deep
            links into the cyber modules that supply evidence, telemetry, remediation, and auditability.
          </p>
        </div>
      </div>
    </section>
  );
}
