# Security Policy

This repository (`Beszel-iOS`, `iOS` branch) is an **unofficial community port** of [Beszel by henrygd](https://github.com/henrygd/beszel) for jailbroken iOS devices. It is not the upstream project.

## What goes where

- **Vulnerability in upstream Beszel itself** (Hub, Agent, frontend, or dependencies affecting all platforms): report to upstream per [henrygd/beszel security policy](https://github.com/henrygd/beszel/blob/main/SECURITY.md). Do not assume this port can fix or backport it.
- **Issue specific to this iOS port** (iOS battery reader, iOS system metadata, runtime patch script, iOS workflows, or iOS packaging/installation): open an issue in this repository with device model, SoC, iOS version, jailbreak/bootstrap, and reproduction steps.

## Reporting notes

- Upstream asks that GitHub Security Advisories be reserved for real high-severity vulnerabilities, with lower-severity findings filed as issues. Follow that guidance for upstream issues.
- For iOS-port issues, prefer a regular issue in this repo; include logs and whether the validated target (iPad mini 2 / A7 / iOS 12.5.7) is involved.
- Do not include secrets, tokens, or private Hub data in reports.

No specific response time or security support level is promised for this community port.
