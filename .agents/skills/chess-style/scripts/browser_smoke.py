#!/usr/bin/env python3
"""Smoke test opcional do starter em Chromium.

Requer Playwright instalado e um Chromium disponível. Não instala software, não
faz chamadas externas e não constitui auditoria completa de acessibilidade.
"""
from __future__ import annotations
import argparse
import functools
import http.server
import json
import shutil
import sys
import threading
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VIEWPORTS = [(1920,1080),(1440,900),(1280,800),(768,1024),(390,844),(360,740),(320,740)]

class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *_args) -> None:
        pass

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--browser", help="Caminho opcional do Chromium local")
    parser.add_argument("--transport", choices=["http", "inline"], default="http", help="inline carrega HTML/CSS/JS em memória, sem navegar para URLs")
    parser.add_argument("--out", type=Path, default=ROOT.parent / "chess-style-checks")
    args = parser.parse_args()
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("Playwright não está instalado. Execute a validação estrutural ou configure o ambiente de teste.", file=sys.stderr)
        return 2
    args.out.mkdir(parents=True, exist_ok=True)
    handler = functools.partial(QuietHandler, directory=str(ROOT))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    url = f"http://127.0.0.1:{server.server_address[1]}/examples/starter/"
    results = {"timeUtc":datetime.now(timezone.utc).isoformat(),"scope":"Chromium local; starter, não referência original", "transport":args.transport, "cases":[], "errors":[]}
    inline_html = (ROOT / "examples/starter/index.html").read_text(encoding="utf-8")
    inline_html = inline_html.replace('<link rel="stylesheet" href="../../tokens/tokens.css">', '<style>' + (ROOT / "tokens/tokens.css").read_text(encoding="utf-8") + '</style>')
    inline_html = inline_html.replace('<link rel="stylesheet" href="styles.css">', '<style>' + (ROOT / "examples/starter/styles.css").read_text(encoding="utf-8") + '</style>')
    inline_html = inline_html.replace('<script src="script.js" defer></script>', '')
    inline_html = inline_html.replace('</body>', '<script>' + (ROOT / "examples/starter/script.js").read_text(encoding="utf-8") + '</script></body>')
    browser_path = args.browser or shutil.which("chromium") or shutil.which("google-chrome")
    try:
        with sync_playwright() as playwright:
            launch_options = {"headless":True}
            if browser_path:
                launch_options["executable_path"] = browser_path
            browser = playwright.chromium.launch(**launch_options)
            results["browserVersion"] = browser.version
            def run_case(width, height, *, mode="default", javascript=True, reduced=False, double_text=False):
                context = browser.new_context(viewport={"width":width,"height":height}, java_script_enabled=javascript,
                    reduced_motion="reduce" if reduced else "no-preference")
                page = context.new_page()
                case = {"viewport":[width,height],"mode":mode,"checks":{},"errors":[]}
                page.on("pageerror", lambda error: case["errors"].append(str(error)))
                page.on("console", lambda message: case["errors"].append(message.text) if message.type == "error" else None)
                if args.transport == "inline":
                    page.set_content(inline_html, wait_until="load")
                    page.wait_for_timeout(100)
                else:
                    page.goto(url, wait_until="networkidle")
                if double_text:
                    page.evaluate("document.documentElement.style.fontSize = '200%'")
                    page.wait_for_timeout(120)
                case["checks"]["oneH1"] = page.locator("h1").count() == 1
                case["checks"]["textPresent"] = len(page.locator("main").inner_text()) > 500
                overflow_positions = []
                for section in ["#inicio", "#ideia", "#metodo", "#conclusao"]:
                    page.locator(section).scroll_into_view_if_needed()
                    page.wait_for_timeout(80)
                    delta = page.evaluate("document.documentElement.scrollWidth - document.documentElement.clientWidth")
                    if delta > 1:
                        overflow_positions.append({"section":section,"pixels":delta})
                case["checks"]["noHorizontalOverflow"] = not overflow_positions
                case["overflow"] = overflow_positions
                if javascript:
                    case["checks"]["enhancedInitialized"] = page.locator("html[data-enhanced]").count() == 1
                    if reduced:
                        case["checks"]["reducedHonored"] = page.locator("html.motion-on").count() == 0 and page.locator(".motion-toggle").is_disabled()
                    elif not double_text:
                        page.locator(".motion-toggle").click()
                        page.wait_for_timeout(100)
                        case["checks"]["staticToggle"] = page.locator(".motion-toggle").get_attribute("aria-pressed") == "true" and page.locator("html.motion-on").count() == 0
                        case["checks"]["staticInk"] = page.evaluate("[...document.querySelectorAll('.reveal-word')].every(x => getComputedStyle(x).color === 'rgb(0, 0, 0)')")
                    if width <= 900 and not double_text:
                        page.locator(".menu-toggle").click()
                        case["checks"]["menuOpens"] = page.locator(".menu-toggle").get_attribute("aria-expanded") == "true"
                        page.keyboard.press("Escape")
                        case["checks"]["escapeReturnsFocus"] = page.evaluate("document.activeElement === document.querySelector('.menu-toggle')") and page.locator(".menu-toggle").get_attribute("aria-expanded") == "false"
                        page.locator(".menu-toggle").click()
                        page.locator(".site-nav a[href='#metodo']").click()
                        case["checks"]["anchorClosesMenu"] = page.locator(".menu-toggle").get_attribute("aria-expanded") == "false" and page.url.endswith("#metodo")
                else:
                    case["checks"]["navigationWithoutJS"] = page.locator(".site-nav").is_visible()
                    case["checks"]["noEmptyReveal"] = page.locator("[data-reveal]").first.inner_text().startswith("Você não precisa")
                page.evaluate("window.scrollTo({top: 0, left: 0, behavior: 'instant'})")
                page.wait_for_function("window.scrollY === 0")
                page.wait_for_timeout(120)
                shot = args.out / f"starter-{width}x{height}-{mode}.png"
                page.screenshot(path=str(shot), full_page=False)
                case["screenshot"] = shot.name
                case["screenshotState"] = {"scrollY":page.evaluate("window.scrollY"), "motionEnabled":page.locator("html.motion-on").count() == 1}
                case["passed"] = all(case["checks"].values()) and not case["errors"]
                results["cases"].append(case)
                context.close()
            for width,height in VIEWPORTS:
                run_case(width,height)
            run_case(390,844,mode="no-js",javascript=False)
            run_case(390,844,mode="reduced",reduced=True)
            run_case(390,844,mode="root-text-200",double_text=True)
            run_case(1440,900,mode="root-text-200",double_text=True)
            browser.close()
    except Exception as exc:
        results["errors"].append(str(exc))
    finally:
        server.shutdown()
        server.server_close()
    results["passed"] = bool(results["cases"]) and all(case["passed"] for case in results["cases"]) and not results["errors"]
    (args.out / "results.json").write_text(json.dumps(results, ensure_ascii=False, indent=2)+"\n", encoding="utf-8")
    print(json.dumps(results, ensure_ascii=False, indent=2))
    return 0 if results["passed"] else 1

if __name__ == "__main__":
    raise SystemExit(main())
