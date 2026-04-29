#!/usr/bin/env python3
"""Generate an HTML proof report with real browser screenshots."""

from __future__ import annotations

import argparse
import html
import os
import shutil
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


def sh(cmd: str, env: dict[str, str], cwd: str | None = None) -> tuple[int, str]:
    proc = subprocess.run(
        cmd,
        shell=True,
        cwd=cwd,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    return proc.returncode, proc.stdout


def find_browser() -> str:
    for name in ("chromium", "chromium-browser", "google-chrome", "google-chrome-stable"):
        path = shutil.which(name)
        if path:
            return path
    raise SystemExit("No headless browser found. Install chromium or google-chrome.")


def wait_for(url: str, timeout: float) -> None:
    deadline = time.time() + timeout
    last_err = None
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2) as resp:
                if 200 <= resp.status < 500:
                    return
        except (urllib.error.URLError, TimeoutError) as exc:
            last_err = exc
        time.sleep(0.25)
    raise RuntimeError(f"Timed out waiting for {url}: {last_err}")


def abs_url(base: str, path_or_url: str) -> str:
    if path_or_url.startswith("http://") or path_or_url.startswith("https://"):
        return path_or_url
    if not path_or_url.startswith("/"):
        path_or_url = "/" + path_or_url
    return base.rstrip("/") + path_or_url


def parse_pipe(value: str, parts: int, label: str) -> list[str]:
    split = value.split("|", parts - 1)
    if len(split) != parts:
        raise SystemExit(f"Invalid {label}: {value!r}. Expected {parts} pipe-separated fields.")
    return [s.strip() for s in split]


def capture(browser: str, url: str, out: Path, window_size: str) -> None:
    cmd = [
        browser,
        "--headless",
        "--no-sandbox",
        "--disable-gpu",
        f"--window-size={window_size}",
        f"--screenshot={out}",
        url,
    ]
    subprocess.run(cmd, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def write_report(
    out_dir: Path,
    title: str,
    summary: str,
    commands: list[dict[str, str | int]],
    evidence: list[tuple[str, str]],
    notes: list[str],
    screenshots: list[dict[str, str]],
) -> None:
    def esc(s: object) -> str:
        return html.escape(str(s), quote=True)

    command_html = "".join(
        f"""
        <div class=\"command {'pass' if c['code'] == 0 else 'fail'}\">
          <div><strong>{'PASS' if c['code'] == 0 else 'FAIL'}</strong> <code>{esc(c['cmd'])}</code></div>
          <pre>{esc(c['output'])}</pre>
        </div>
        """
        for c in commands
    )
    evidence_html = "".join(
        f"<div class=\"check\"><strong>{esc(k)}</strong><span>{esc(v)}</span></div>" for k, v in evidence
    )
    notes_html = "".join(f"<li>{esc(n)}</li>" for n in notes)
    shots_html = "".join(
        f"""
        <article class=\"shot\">
          <div class=\"shot-body\">
            <h3>{esc(s['title'])}</h3>
            <p>{esc(s['description'])}</p>
            <code>{esc(s['url'])}</code>
          </div>
          <img src=\"{esc(s['src'])}\" alt=\"{esc(s['title'])} screenshot\" />
        </article>
        """
        for s in screenshots
    )

    doc = f"""<!DOCTYPE html>
<html lang=\"en\">
<head>
<meta charset=\"utf-8\" />
<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />
<title>{esc(title)}</title>
<style>
:root{{--bg:#f8f7f4;--surface:#fff;--line:#e6e3de;--text:#1a1816;--muted:#7f786f;--primary:#3b4a5c;--accent:#c1754a;--ok:#3f7d55;--danger:#b54a3e;--shadow:0 1px 3px rgba(26,24,22,.04),0 12px 32px rgba(26,24,22,.07);--sans:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;--mono:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}}
*{{box-sizing:border-box}}body{{margin:0;background:var(--bg);color:var(--text);font:15px/1.6 var(--sans)}}header{{position:sticky;top:0;z-index:10;background:rgba(248,247,244,.92);backdrop-filter:blur(14px);border-bottom:1px solid var(--line)}}.top{{max-width:1160px;margin:auto;padding:14px 20px;display:flex;align-items:center;justify-content:space-between;gap:16px}}.brand{{font-weight:850;letter-spacing:-.035em}}.brand small{{display:block;color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.14em}}main{{max-width:1160px;margin:auto;padding:28px 20px 70px}}.hero{{display:grid;grid-template-columns:1.15fr .85fr;gap:18px}}.card,.shot{{background:var(--surface);border:1px solid var(--line);border-radius:16px;box-shadow:var(--shadow)}}.card{{padding:20px}}h1{{font-size:40px;line-height:1.05;letter-spacing:-.06em;margin:0 0 12px}}h2{{font-size:22px;letter-spacing:-.035em;margin:0 0 12px}}h3{{font-size:16px;margin:0 0 6px}}p{{color:var(--muted);margin:0 0 10px}}code,pre{{font-family:var(--mono)}}code{{background:#f1efeb;border-radius:5px;padding:2px 5px;color:#29303a}}pre{{background:#202734;color:#eef2f6;border-radius:12px;padding:12px;overflow:auto;font-size:12px;max-height:280px}}.grid{{display:grid;grid-template-columns:repeat(2,1fr);gap:18px;margin-top:18px}}.checks{{display:grid;gap:10px}}.check{{display:grid;grid-template-columns:180px 1fr;gap:12px;color:var(--muted);border-top:1px solid var(--line);padding-top:10px}}.check strong{{color:var(--text)}}.command{{border-left:4px solid var(--ok);background:#fbfaf8;border-radius:12px;padding:12px;margin-bottom:10px}}.command.fail{{border-left-color:var(--danger)}}.shot{{overflow:hidden}}.shot-body{{padding:16px}}.shot img{{width:100%;display:block;border-top:1px solid var(--line);background:#fff}}.tag{{display:inline-block;border-radius:999px;background:#eef1f4;color:var(--primary);font-weight:850;font-size:12px;padding:3px 8px}}ul{{color:var(--muted)}}@media(max-width:850px){{.hero,.grid{{grid-template-columns:1fr}}h1{{font-size:31px}}.check{{grid-template-columns:1fr}}.top{{align-items:flex-start;flex-direction:column}}}}
</style>
</head>
<body>
<header><div class=\"top\"><div class=\"brand\">{esc(title)}<small>Generated proof report</small></div><span class=\"tag\">real screenshots</span></div></header>
<main>
  <section class=\"hero\">
    <div class=\"card\"><h1>{esc(title)}</h1><p>{esc(summary)}</p><div class=\"checks\">{evidence_html}</div></div>
    <div class=\"card\"><h2>Commands run</h2>{command_html}</div>
  </section>
  {f'<section class="card" style="margin-top:18px"><h2>Notes / limitations</h2><ul>{notes_html}</ul></section>' if notes_html else ''}
  <section class=\"grid\">{shots_html}</section>
</main>
</body>
</html>
"""
    (out_dir / "index.html").write_text(doc, encoding="utf-8")


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--title", required=True)
    ap.add_argument("--summary", default="Feature proof report generated from a running app.")
    ap.add_argument("--out", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--start-command", help="Shell command to start the app. If omitted, assumes app is already running.")
    ap.add_argument("--health-path", default="/")
    ap.add_argument("--timeout", type=float, default=30)
    ap.add_argument("--window-size", default="1280,1000")
    ap.add_argument("--verify-command", action="append", default=[])
    ap.add_argument("--setup-command", action="append", default=[])
    ap.add_argument("--screenshot", action="append", default=[], help="title|path-or-url|description")
    ap.add_argument("--evidence", action="append", default=[], help="label|value")
    ap.add_argument("--implementation-note", action="append", default=[])
    args = ap.parse_args()

    out_dir = Path(args.out)
    screenshot_dir = out_dir / "screenshots"
    out_dir.mkdir(parents=True, exist_ok=True)
    screenshot_dir.mkdir(parents=True, exist_ok=True)

    env = os.environ.copy()
    env.update({
        "BASE_URL": args.base_url.rstrip("/"),
        "REPORT_DIR": str(out_dir),
        "SCREENSHOT_DIR": str(screenshot_dir),
    })

    commands: list[dict[str, str | int]] = []
    proc: subprocess.Popen[str] | None = None
    try:
        for cmd in args.verify_command:
            code, output = sh(cmd, env)
            commands.append({"cmd": cmd, "code": code, "output": output})
            if code != 0:
                raise RuntimeError(f"Verification command failed: {cmd}")

        if args.start_command:
            log = open(out_dir / "server.log", "w", encoding="utf-8")
            proc = subprocess.Popen(
                args.start_command,
                shell=True,
                env=env,
                text=True,
                stdout=log,
                stderr=subprocess.STDOUT,
                preexec_fn=os.setsid if hasattr(os, "setsid") else None,
            )
            wait_for(abs_url(args.base_url, args.health_path), args.timeout)

        for cmd in args.setup_command:
            code, output = sh(cmd, env)
            commands.append({"cmd": cmd, "code": code, "output": output})
            if code != 0:
                raise RuntimeError(f"Setup command failed: {cmd}")

        browser = find_browser()
        shots: list[dict[str, str]] = []
        for i, raw in enumerate(args.screenshot, start=1):
            title, path, desc = parse_pipe(raw, 3, "--screenshot")
            url = abs_url(args.base_url, path)
            filename = f"{i:02d}-" + "".join(c.lower() if c.isalnum() else "-" for c in title).strip("-") + ".png"
            capture(browser, url, screenshot_dir / filename, args.window_size)
            shots.append({
                "title": title,
                "description": desc,
                "url": url,
                "src": "screenshots/" + filename,
            })

        evidence = [tuple(parse_pipe(e, 2, "--evidence")) for e in args.evidence]
        write_report(out_dir, args.title, args.summary, commands, evidence, args.implementation_note, shots)
        print(out_dir / "index.html")
        return 0
    finally:
        if proc and proc.poll() is None:
            if hasattr(os, "killpg"):
                os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
            else:
                proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()


if __name__ == "__main__":
    raise SystemExit(main())
