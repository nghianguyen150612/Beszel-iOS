# P003 risk register method

[`upstream-v021-risk-register.csv`](upstream-v021-risk-register.csv) is the
machine-readable register. It contains one row for every P002 P0 SHA plus
database, iOS, installer, frontend, dependency, and remote-platform risks. The
13 logical iOS patch IDs are cross-referenced across the register and the
implementation crosswalk.

## Deterministic ordinal score

Each risk receives an ordinal severity and likelihood before applying the
listed mitigation. The numeric score is:

`risk_score = severity_value × likelihood_value`

| Severity | Value | Consequence if the failure occurs |
| --- | ---: | --- |
| Critical | 5 | Data loss, cross-user exposure, privilege boundary failure, or complete service/build failure on the supported A7 target. |
| High | 4 | Major loss of telemetry, authorization, history, or upgrade/recovery capability. |
| Moderate | 3 | A material feature or metric is unavailable or incorrect with a bounded workaround. |
| Low | 2 | Limited degradation with little data or security impact. |
| Negligible | 1 | Cosmetic or readily reversible effect. |

| Likelihood | Value | Ordinal meaning for this planned integration |
| --- | ---: | --- |
| Rare | 1 | Requires an unusual condition and multiple safeguards to fail. |
| Unlikely | 2 | Needs a specific trigger, but is plausible. |
| Possible | 3 | A credible trigger exists in the integration or deployment path. |
| Likely | 4 | Source evidence or workflow makes the trigger expected unless controlled. |
| Almost certain | 5 | Failure follows directly from the current design or is already reproducible. |

Scores 1–4 are lower priority, 5–9 moderate, 10–16 high, and 17–25 critical
for sequencing review. These are planning estimates, **not observed failure
rates, incident frequencies, or statistical probabilities**. Detectability is
recorded separately (`High`, `Medium`, `Low`) and is not multiplied into the
score.

## Status classes

* `confirmed incompatibility` — source inspection establishes a missing or
  conflicting iOS behavior, such as an upstream Darwin implementation that
  cannot serve as an iOS adapter. The status describes the source contract, not
  a measured production incident.
* `potential regression` — a concrete failure mode could result from integrating
  the source change or from an interaction with a preserved iOS patch.
* `untested risk` — no completed test establishes the behavior, and the source
  evidence alone cannot decide whether the risk will occur.

Every row names a mitigation, required test, owner prompt, blocking gate, and
rollback/containment action. A critical data-loss or security risk without an
effective mitigation and blocking gate prevents the relevant gate from passing.
