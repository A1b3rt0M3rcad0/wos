# Licensing and brand

Current first-party material offered under the [WOS Sustainable Use License 1.0](../LICENSE) is source-available. It permits production use and commercial application infrastructure while withholding commercial supply of the platform itself. It is not an OSI Open Source or unrestricted permissive license.

This guide explains the policy; the applicable license text governs. Brand rights are separate under [TRADEMARKS.md](../TRADEMARKS.md).

## What you can do

| Use | Permission under the current license |
| --- | --- |
| Run WOS in your company's production environment | Permitted for your own internal operations, on your infrastructure or rented general-purpose infrastructure. |
| Coordinate your coding agents and human reviewers | Permitted; WOS records and coordinates work while your runtimes perform it. |
| Build a paid application or Outcome-as-a-Service product | Permitted when you deliver your own application's functionality or customer outcomes, with WOS as its infrastructure. |
| Install your application on a customer's infrastructure | Permitted with WOS as a component, preserving the WOS terms and notices for the included material. |
| Charge for setup, integration, training, maintenance or support | Permitted for genuine technical work on the customer's own permitted deployment; the fee cannot be for supplying a WOS instance or continued platform access. |
| Keep private modifications | Permitted for authorized uses, without a requirement to publish private modifications or independently developed application code. |
| Share a free non-commercial fork | Permitted with the required notices, independent-origin identification and brand rules. |

An application is not excluded merely because it depends heavily on WOS, uses its APIs, shows embedded progress or provides configuration limited to its own purpose.

## What the license does not grant

You cannot commercially supply the WOS platform or a material part of its general-purpose coordination capabilities. This includes selling copies or appliances, supplying paid hosted/managed instances, charging for general-purpose workspace/API access, OEM supply and commercial white-label derivatives.

The restriction applies under the original brand, a different name, a redesigned interface or a wrapper. It also covers indirect charges and paid bundles that label the WOS component free. Genuine permitted applications and technical services remain permitted; calling platform access a support service does not change what is supplied.

| Example | Boundary |
| --- | --- |
| A paid service implements and verifies a requested feature, using WOS to track its workers and proof | Permitted application/outcome delivery. |
| A support engineer charges for configuring a customer's self-hosted WOS deployment | Permitted technical services when the customer controls the deployment and the fee is for the work. |
| A vendor charges monthly for access to a WOS workspace or a dedicated WOS instance | Prohibited commercial platform supply. |
| A vendor renames WOS and sells the same general-purpose coordination backend | Prohibited rebranded/white-label platform supply. |
| A paid bundle includes WOS platform access at no separately listed price | Still prohibited; the bundle does not remove the commercial consideration. |

## Brand and attribution

Legitimate installations may display their original branding. Truthful secondary references such as "built with WOS" or "compatible with WOS" are permitted when accurate and non-misleading. They do not imply official representation, partnership, endorsement or reseller status.

Permitted internal interfaces and your own applications may use their own presentation while retaining legal notices. That does not authorize commercial platform resale under a different brand. Forks must identify independent origin and material changes; retaining original branding does not make them official distributions.

The [trademark policy](../TRADEMARKS.md) makes no claim that a mark is registered in a particular jurisdiction.

## Earlier releases and third-party rights

The repository previously used Apache-2.0, including published v0.4.2 and earlier source revisions. Those grants remain effective for the material they cover, including portions retained in later distributions. This change does not revoke earlier rights, rewrite published artifacts or automatically place new source-available-only material under Apache. The original text remains in [LICENSES/Apache-2.0.txt](../LICENSES/Apache-2.0.txt).

Third-party licenses, including dependency and font licenses, retain their own permissions and notice obligations. [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) describes the inventory. Restrictions in the new license do not narrow independently granted rights in those materials. A file replacement alone does not establish ownership or authority to relicense another party's contribution.

## Distribution and contributions

Permitted distributions must retain the current license, trademark policy, notices, preserved Apache text and applicable upstream notices. These travel with source, native/npm/container artifacts, installed Skills and embedded copies. The [release guide](releases.md) explains the packaging paths; generated dependency notices remain required.

New first-party contributions are offered under the repository's current license as described in [CONTRIBUTING.md](../CONTRIBUTING.md). Contributors retain ownership and must have authority for what they submit. This policy does not retroactively obtain rights over existing contributions. [Technical verification](licensing-validation.md) records artifact and installed-notice checks separately from legal review.

## Decision and reference

The owner requested the production/application permissions and platform-resale/brand limits used by Woobe. The reference was [Woobe's license](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/LICENSE.md) and [trademark policy](https://github.com/A1b3rt0M3rcad0/woobe/blob/3dbcb3e1a7559f4a50ebca320e411933d46e8012/TRADEMARKS.md) at that pinned revision. WOS retains independent terms, distribution and runtime boundaries. [ADR 037](adr/0037-source-available-licensing-and-brand-boundaries.md) records the decision and supersedes the original Open Source direction for new material lawfully supplied under the current text.

This is an implementation of the owner's policy, not a legal opinion about enforceability or title to every contribution. Qualified legal review of the text and contribution rights is recommended before relying on the new terms for public commercial distribution.
