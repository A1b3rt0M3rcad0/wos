import { copy } from "./en-US.js";

// Decorative artwork and editorial introductions share the existing area state.
// They never derive business status, permissions or suggested commands.
const chapters = Object.freeze({
  overview: [
    "01",
    "wholePicture",
    "overviewEditorial",
    "sharedByDesign",
    "14_observer_metrics.svg",
  ],
  plan: [
    "02",
    "planWithPurpose",
    "planEditorial",
    "aMapNotAMandate",
    "01_planner_map.svg",
  ],
  work: [
    "03",
    "makeWorkMove",
    "workEditorial",
    "oneStepAtATime",
    "05_coordinator_kanban.svg",
  ],
  issues: [
    "04",
    "clearTheWay",
    "issuesEditorial",
    "makeTheFrictionVisible",
    "16_blocker_sentinel.svg",
  ],
  evidence: [
    "05",
    "proofBeforeConclusions",
    "evidenceEditorial",
    "showYourWork",
    "06_researcher_books.svg",
  ],
  activity: [
    "06",
    "everyStepRemembered",
    "activityEditorial",
    "aThreadWorthKeeping",
    "09_documenter_notes.svg",
  ],
});

export function renderEditorial(area) {
  const [index, title, description, note, robot] = chapters[area];
  document.getElementById("area-index").textContent = `${index} / WOS`;
  document.getElementById("area-title").textContent = copy.text[title];
  document.getElementById("area-description").textContent =
    copy.text[description];
  document.getElementById("area-note").textContent = copy.text[note];
  document.getElementById("area-robot").src = `robots/${robot}`;
  document.getElementById("outcome").dataset.editorialArea = area;
}

export function sectionIllustration(area) {
  const img = document.createElement("img");
  img.src = `robots/${chapters[area][4]}`;
  img.alt = "";
  img.width = 112;
  img.height = 112;
  img.className = "section-vignette";
  return img;
}
