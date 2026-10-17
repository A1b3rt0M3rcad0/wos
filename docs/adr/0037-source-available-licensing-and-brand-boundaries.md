# ADR 037: Source-available licensing and brand boundaries

Status: Accepted by explicit owner instruction on 2026-10-17.

## Context

The repository used Apache-2.0. The owner now requests production use and commercial application infrastructure with restrictions against selling the platform, supplying paid platform access, or commercially white-labeling its technology under any brand. The requested reference is Woobe's software and trademark policy.

The original canonical specification describes an Open Source licensing direction. That historical source document remains unchanged. This decision supersedes that direction for material lawfully distributed under the new license; it does not change domain invariants, deployment independence or API behavior.

## Decision

Adopt the WOS Sustainable Use License 1.0 and a separate trademark policy. Current first-party source is source-available, not OSI Open Source. Permit internal production, self-hosting, private modification, commercial applications/customer outcomes, component distribution within permitted applications, genuine technical services and free non-commercial distribution within the stated conditions.

Withhold commercial supply of the WOS platform itself, including general-purpose coordination capability, paid hosted/managed instances, resale, OEM and commercial white-label derivatives. Original branding, a new brand, a wrapper or bundled pricing does not change that restriction. Preserve ordinary installation identification and truthful descriptive references without conferring official status.

Use `SEE LICENSE IN LICENSE` in npm metadata and `LicenseRef-WOS-Sustainable-Use-1.0` for the OCI license identifier. Carry the license, trademark policy, notices, prior Apache text and upstream notices through distribution and Skill installation.

## Consequences

- The [licensing guide](../licensing.md) explains application/OaaS permissions separately from platform supply.
- Earlier Apache grants and separately licensed material retain their rights. Historical releases and immutable artifacts are not relabeled. Changing a root license does not revoke earlier permissions or obtain rights to other owners' material.
- Contribution authority remains explicit; no automatic copyright assignment or retroactive relicensing is introduced.
- Third-party dependency, font and other notices remain intact; generated artifact notices are additional to the first-party policy.
- This decision does not publish a new release, deploy software or add commercial service capabilities. Legal review is distinct from technical packaging validation.

## Reference

Woobe [LICENSE.md](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/LICENSE.md) and [TRADEMARKS.md](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/TRADEMARKS.md), inspected at commit `3dbcb3e1a7559f4a50ebca320e411933d46e8012`.
