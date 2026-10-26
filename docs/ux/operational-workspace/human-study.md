# Independent comprehension study

**Status: not run. No participants or usability results are recorded.**

This is the executable study protocol for the [owner's human-comprehension
requirements](owner-plan-2026-10-19.md#6-teste-de-compreensão-humana).
Use the [blank results template](human-study-results.template.json) to record an
actual session. Automated tests, screenshots and an author's walkthrough remain
separate evidence in [qualification](qualification.md).

## Participants and preparation

Recruit at least five people who did not participate in designing this interface.
Record that independence, their familiarity with agent-assisted development and
their prior WOS experience. Use participant codes rather than names. Identify the
first five eligible participants as the primary cohort before examining results;
report additional sessions separately. Do not replace a failed or abandoned
attempt with a more successful participant.

Use an isolated study installation with disposable content and separate
participant accounts. Record the exact source revision, server build, adapter,
protocol, browser/version, CSS viewport, device, native zoom and input method.
Record fixture preparation and any deviations. These sessions do not qualify
unmeasured platforms or production latency.

The observer prepares each starting state through supported product operations.
Do not fabricate execution, review or acceptance by changing database rows or
assigning a display label. A signed fixture needs genuinely distinct executor
and reviewer identities and their valid recorded contracts. If a required fixture
cannot be prepared, mark the corresponding task **not run**, with the reason.

Give participants the supplied content below in a separate readable sheet. It is
fictional study material, not a claim that a participant ran a real installation.
Do not teach internal command names, show IDs, provide console instructions or
explain the distinction being tested before an attempt. Offer no tutorial of the
controls. Read only this introduction:

> This workspace records a team's work and the information used to review its
> result. Please carry out each request using the interface. The content is
> supplied so you do not need to invent a plan. You may stop or ask for help.

## Timing and observation

Read the task and content before timing. Start when the participant says they are
ready and the specified starting screen is available. Stop at the first correctly
completed operation or answer. Record elapsed time at an error, assistance or
abandonment as well as at completion. Do not require continuous think-aloud speech
during the short timed tasks; ask for reasoning afterwards.

Do not pause the clock for interface searches, retries or correction of mistakes.
Pause only for an unrelated interruption and record its cause and duration. If
the participant requests help, record their question and the exact assistance
given. Keep assisted completion separate from unassisted completion. Continue
observing after the target time when useful, retaining the original target miss.
There is no invented time limit for the two conceptual tasks.

For every attempt record the starting fixture, time, final persisted result or
spoken answer, wrong targets, mistaken actions, recovery, assistance, abandonment
and the UI text that may have induced a misunderstanding. Never enter a guessed
completion time for an abandoned task.

## H01 — Build a small plan

**Start:** the participant is signed in to an empty study workspace.

**Participant request:**

> Record the following result and its work. Keep the work planned; nobody should
> start executing it yet. Include the supplied success conditions. Stop when the
> complete plan is saved and you can see it in the workspace.

| Item | Supplied content | Belongs to |
| --- | --- | --- |
| Outcome | **Publish usable installation instructions** | Study workspace |
| Desired result | A new teammate can start a local workspace using the published instructions. | Outcome |
| Outcome success condition | A teammate starts a usable workspace using only the published instructions. | Outcome; required |
| Objective | **Setup instructions work** | Outcome; required for the Outcome |
| Success condition | A clean test environment starts with the documented setup steps. | Setup instructions work; required |
| Task | **Try setup on a clean computer** | Setup instructions work |
| Task | **Record the installation output** | Setup instructions work |
| Objective | **Instructions receive an independent review** | Outcome; required for the Outcome |
| Success condition | A reviewer confirms the instructions contain every required setup step. | Instructions receive an independent review; required |
| Task | **Check every installation step** | Instructions receive an independent review |
| Task | **Review the final instructions** | Instructions receive an independent review |

**Observer checks:** one draft Outcome, two planned Objectives, four backlog
Tasks and three explicit required Criteria are persisted under the intended
parents. No execution authority is acquired and no assessment or acceptance is
invented. Keep the ordinary visible evidence-review suggestion for these Criteria.

**Owner threshold:** median completion time at most **180 seconds**, without
entering IDs or using a technical console. Record all five attempts, including
noncompletion; a median calculated from successful attempts alone must be labeled
with its denominator and cannot silently stand for the whole cohort.

## H02 — Read a Task

**Start:** a prepared Outcome Board is open; no Task inspector is open. The Task
**Review installation on a clean machine** has these canonical facts:

- Instruction: “Follow the installation guide on the supplied clean test environment.”
- Deliverable: “An installation transcript with the commands and their results.”
- Casey is the declared assignee; no one holds current execution authority.
- Required Criterion: **Clean setup starts a usable workspace**; not assessed.
- A supporting Evidence record is linked to that Criterion.
- A documentary update says “I recorded the first installation attempt.” No
  delivery has been submitted; the Task remains planned in backlog.

**Participant request:**

> Open “Review installation on a clean machine.” Show or tell me: (1) the work
> requested, (2) its success condition and supporting information, (3) who is
> responsible and who currently has execution authority, (4) what has been
> recorded or delivered, and (5) what still needs to happen next. Stay in this
> workspace view.

**Observer checks:** all five groups are found in the same Task context. The
participant recognizes that an update and linked Evidence do not mean a delivery
or a met Criterion; backlog is planned work, not acquired execution. Do not score
knowledge of internal status identifiers.

**Owner threshold:** at least **4 of the primary 5** complete this within
**30 seconds**, without switching the global view.

## H03 — Add information to the correct condition

**Start:** the H02 Task inspector is open. The named Criterion is current and
unassessed. Reset to this same state for each participant.

**Participant request:**

> Add this supporting information to “Clean setup starts a usable workspace,”
> using the supplied reference. Do not make an acceptance decision.

- Observation: **The supplied study transcript records that the setup command opened the login screen in the test environment.**
- Reference: **https://example.test/study/installation-transcript**
- Relevance: **The transcript records the environment and startup result for this condition.**

**Observer checks:** one Evidence record and its supporting link are saved on
that exact Criterion and its Task. No link is lost or redirected to another
owner. The Criterion remains unassessed and the Task lifecycle is unchanged.

**Owner threshold:** at least **4 of 5** within **45 seconds**, with the correct
Criterion link retained. Record assistance and mistaken targets separately.

## H04 — Report an impediment deliberately

**Start:** a separate, eligible Task **Check deployment access** is open under
**Setup instructions work**. It needs access to the test environment to continue.
No Blocker already applies to it.

**Participant request:**

> Nobody can access the test environment needed to continue this Task. Record the
> problem on “Check deployment access” and decide whether the work must wait.
> This affects this Task; do not stop unrelated work in the Outcome.

- Problem: **Test environment access is unavailable**
- Description: **The supplied study account cannot access the environment required for this check.**
- Severity: **Major**
- Impact: **This access check must wait until the study account can enter the test environment.**

**Observer checks:** the Issue identifies the selected Task, and an explicit
direct Blocker records that Task's impediment. No unrelated branch is blocked.
Ask afterwards which other work is affected and why. Creating an Issue alone is
not the required blocking decision in this supplied situation.

**Owner threshold:** at least **4 of 5** within **45 seconds**, with correct
affected item and blocking impact.

## H05 — Distinguish responsibility and authority

**Start:** the observer opens the prepared Task **Review the installer delivery**.
Casey is its declared assignee. Alex submitted its latest execution delivery;
that execution contract no longer grants current execution authority. Morgan
holds the valid current independent-review authority. Drew authored a separate
documentary update. The delivery awaits review; the Task is not done.

**Participant request:**

> Who is listed as responsible for this Task? Who submitted the last delivery?
> Who is authorized to review it now? Does Casey's assignment or Drew's update
> authorize either person to execute or approve the work? Does this screen tell
> you whether any of them is online right now?

**Observer answer key:** Casey is the declared assignee; Alex is the recorded
executor of the submitted delivery; Morgan has current review authority. An
assignment, authorship or claimed progress does not grant execution/review rights.
Recorded authority is not proof of a running process or online presence. Do not
describe the delivered execution contract as current execution authority.

**Owner threshold:** **no error that grants fictitious authority or confuses
authorship**. Record verbatim answers, critical/noncritical mistakes and whether
the misunderstanding was induced by the UI. There is no owner-specified time cap.

## H06 — Distinguish information, verification and acceptance

**Start:** prepare real, separately identifiable records with these facts, and
open each requested context when the participant is ready:

1. **Startup transcript:** linked as supporting Evidence to an unassessed Criterion.
2. **Installation steps checked:** a current required Objective Criterion has a
   valid “met” assessment; the Objective itself is still planned, not achieved.
3. **Installer delivery awaiting review:** execution material was received; its
   independent review is pending and its Task is not completed.

**Participant request:**

> What can we conclude from each record? Has linking the transcript met its
> condition? Has meeting the Objective's condition accepted the Objective or the
> whole Outcome? Has receiving the delivery accepted its quality or completed
> the Task? Point to the information supporting your answers.

**Observer answer key:** the link records relevance, not an assessment. The
current “met” assessment applies to its specific Criterion; it does not silently
achieve the Objective or the Outcome. Received material is not an approved review
or completed Task. Assessments, delivery, review and explicit conclusion are
separate records. Do not credit an explanation based on an old Criterion revision
or old submission as current proof.

**Owner threshold:** **no critical misunderstanding induced by the UI** during
this script. Record the participant's actual words and the relevant visible
wording, including contradictory evidence. There is no owner-specified time cap.

## Reporting

Copy the JSON template to a new dated results file; leave the original blank.
Keep `participants` empty and `status` as `not_run` until real sessions occur.
Fill source/environment/fixture details, then add one observed participant entry
per session using `participant_entry_template`. Do not put that template itself
in the actual participant array.

Report each owner threshold separately: the H01 median and its complete
denominator; H02/H03/H04 successes within their exact limits out of the primary
five; and every authority/authorship or proof/acceptance misunderstanding from
H05/H06. Show assisted, late, failed, abandoned and unrun attempts explicitly.
Do not invent missing times or treat missing answers as correct. A smaller cohort
cannot satisfy the five-participant requirement.

Five sessions identify problems; they do not estimate market-wide satisfaction.
If independent participants are unavailable, record an owner walkthrough as such
in a separate artifact and leave independent validation **pending**. Any decision
to advance the PR with this study outstanding belongs in the final delivery
report; preparation of this protocol does not authorize a merge or release.
