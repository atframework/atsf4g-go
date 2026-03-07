"""Generate Markdown documentation from .proto files.

Sources:
  - src/lobbysvr/protocol/public
  - src/component/protocol/public

Destination:
    - doc/docs/protocols

Requirement:
    - Preserve directory structure relative to proto roots.
    - For src/component/protocol/public: strip the first-level directory name
        from generated output paths (e.g. pbdesc/protocol/pbdesc -> protocol/pbdesc).

This script is intended to run on Windows/macOS/Linux.
It uses protoc + protoc-gen-doc.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path


def _escape_md_table_cell_text(s: str) -> str:
    # Escape characters that would break markdown table structure.
    return s.replace("|", "\\|")


def _append_to_md_table_row(row: str, addition: str) -> str:
    """Append content to the last cell of a markdown table row.

    If the row already ends with a trailing pipe, we insert before it.
    Otherwise we append directly.
    """

    if row.rstrip().endswith("|"):
        idx = row.rfind("|")
        if idx >= 0:
            return row[:idx] + addition + row[idx:]
    return row + addition


def normalize_markdown_tables(md_file: Path) -> None:
    """Repair markdown tables that are broken by multiline cells.

    Some generators (e.g. protoc-gen-doc) may emit enum/message descriptions that
    contain newlines or blank lines. In vanilla Markdown tables, a row cannot
    span multiple lines; this breaks rendering.

    Strategy: when inside a table, treat non-row lines as a continuation of the
    previous row's last cell (append with <br>). Blank lines become <br><br>.
    """

    if not md_file.is_file():
        return

    original = md_file.read_text(encoding="utf-8")
    lines_in = original.splitlines()
    lines_out: list[str] = []

    in_table = False
    expected_pipes = 0
    cur_row: str | None = None
    pending_parabreak = False

    i = 0
    while i < len(lines_in):
        line = lines_in[i]

        if not in_table:
            # Detect a markdown table by header + separator line.
            if line.lstrip().startswith("|") and i + 1 < len(lines_in):
                nxt = lines_in[i + 1]
                if nxt.lstrip().startswith("|") and "---" in nxt:
                    in_table = True
                    expected_pipes = line.count("|")
                    cur_row = None
                    lines_out.append(line.rstrip())
                    lines_out.append(nxt.rstrip())
                    i += 2
                    continue

            lines_out.append(line.rstrip())
            i += 1
            continue

        # in_table
        if line.lstrip().startswith("|"):
            if cur_row is not None:
                if not cur_row.rstrip().endswith("|"):
                    cur_row = cur_row.rstrip() + " |"
                lines_out.append(cur_row.rstrip())
            cur_row = line.rstrip()
            pending_parabreak = False
            i += 1
            continue

        if cur_row is None:
            # Unexpected: table without any row yet.
            in_table = False
            expected_pipes = 0
            continue

        # If current row is complete, any non-row line ends the table.
        row_complete = cur_row.count(
            "|") >= expected_pipes and cur_row.rstrip().endswith("|")
        if row_complete:
            lines_out.append(cur_row.rstrip())
            cur_row = None
            in_table = False
            expected_pipes = 0
            pending_parabreak = False
            # Re-process this line outside table.
            continue

        stripped = line.strip()
        if stripped == "":
            # Blank line inside an incomplete row -> paragraph break.
            cur_row = _append_to_md_table_row(cur_row, "<br><br>")
            pending_parabreak = True
            i += 1
            continue

        # Continuation of an incomplete row. Preserve trailing delimiter if present.
        prefix = "" if pending_parabreak else "<br>"
        pending_parabreak = False
        if stripped.endswith("|"):
            body = stripped[:-1].rstrip()
            cur_row = _append_to_md_table_row(
                cur_row,
                prefix + _escape_md_table_cell_text(body) + " |",
            )
        else:
            cur_row = _append_to_md_table_row(
                cur_row,
                prefix + _escape_md_table_cell_text(stripped),
            )
        i += 1

    if cur_row is not None:
        if not cur_row.rstrip().endswith("|"):
            cur_row = cur_row.rstrip() + " |"
        lines_out.append(cur_row.rstrip())

    updated = "\n".join(lines_out).rstrip() + "\n"
    if updated != original:
        md_file.write_text(updated, encoding="utf-8")


def remove_scalar_value_types_section(md_file: Path) -> None:
    """Remove protoc-gen-doc's 'Scalar Value Types' section.

    The generated section contains generic notes like:
      - "Uses variable-length encoding..."
      - "A string must always contain UTF-8..."
    which read like lint noise in our docs.
    """

    if not md_file.is_file():
        return

    original = md_file.read_text(encoding="utf-8")
    lines = original.splitlines()
    out: list[str] = []

    skipping = False
    for line in lines:
        # Drop TOC entry.
        if "[Scalar Value Types]" in line or "(#scalar-value-types)" in line:
            continue

        if not skipping and line.strip() == "## Scalar Value Types":
            skipping = True
            continue

        if skipping:
            # Stop skipping at the separator that precedes our plugin section,
            # or any other new top-level section.
            stripped = line.strip()
            if stripped == "---" or stripped.startswith("## "):
                skipping = False
                out.append(line.rstrip())
            continue

        out.append(line.rstrip())

    updated = "\n".join(out).rstrip() + "\n"
    if updated != original:
        md_file.write_text(updated, encoding="utf-8")


def _remove_proto_line_comment(line: str) -> str:
    """Remove // comments from a .proto source line, respecting quoted strings."""

    in_str = False
    escaped = False
    for i in range(len(line) - 1):
        ch = line[i]
        if escaped:
            escaped = False
            continue
        if ch == "\\":
            escaped = True
            continue
        if ch == '"':
            in_str = not in_str
            continue
        if not in_str and ch == "/" and line[i + 1] == "/":
            return line[:i]
    return line


def _unescape_proto_string(s: str) -> str:
    # Minimal unescape for proto string literals.
    return (s.replace("\\\\",
                      "\\").replace("\\\"", '"').replace("\\n", "\n").replace(
                          "\\r", "\r").replace("\\t", "\t"))


_RE_ENUM_START = re.compile(
    r"^\s*enum\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*\{")
_RE_ENUM_VALUE = re.compile(
    r"^\s*(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?P<num>-?\d+)")
_RE_XRES_ENUM_ALIAS = re.compile(
    r"\(\s*org\.xresloader\.enum_alias\s*\)\s*=\s*\"(?P<alias>(?:\\.|[^\"\\])*)\""
)
_RE_XRES_FIELD_ALIAS = re.compile(
    r"\(\s*org\.xresloader\.field_alias\s*\)\s*=\s*\"(?P<alias>(?:\\.|[^\"\\])*)\""
)
_RE_ERROR_CODE_DESC = re.compile(
    r"\(\s*error_code\.description\s*\)\s*=\s*\"(?P<desc>(?:\\.|[^\"\\])*)\"")
_RE_ERROR_CODE_SHOW = re.compile(
    r"\(\s*error_code\.show_code\s*\)\s*=\s*(?P<flag>true|false)")
_RE_ATTRIBUTE_ENUM_MODE = re.compile(
    r"\(\s*(?:proy\.)?attribute_enum_extension\s*\)\s*=\s*\{\s*mode\s*:\s*(?P<mode>[A-Za-z_][A-Za-z0-9_]*)\s*\}"
)

_RE_MESSAGE_START = re.compile(
    r"^\s*message\s+(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*\{")
_RE_FIELD_DECL = re.compile(
    r"\b(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?P<num>\d+)\s*(?:\[|;)")


def _extract_enum_value_plugin_info(
        proto_text: str) -> dict[str, list[dict[str, object]]]:
    """Extract per-enum-value plugin metadata from .proto source.

    We currently care about:
      - (org.xresloader.enum_alias) string on EnumValueOptions
      - (error_code.description) string on EnumValueOptions
      - (error_code.show_code) bool on EnumValueOptions

    Note: We parse source text rather than descriptors to avoid requiring
    compiled Python descriptors for custom extensions.
    """

    enums: dict[str, list[dict[str, object]]] = {}

    cur_enum: str | None = None
    brace_depth = 0
    stmt_buf: list[str] = []

    for raw_line in proto_text.splitlines():
        line = _remove_proto_line_comment(raw_line).rstrip()
        if not line.strip():
            continue

        if cur_enum is None:
            m = _RE_ENUM_START.match(line)
            if m:
                cur_enum = m.group("name")
                enums.setdefault(cur_enum, [])
                brace_depth = 1
                # Count any additional braces on same line.
                brace_depth += line.count("{") - 1
                brace_depth -= line.count("}")
            continue

        # Inside enum block
        brace_depth += line.count("{")
        brace_depth -= line.count("}")

        if brace_depth <= 0:
            cur_enum = None
            brace_depth = 0
            stmt_buf.clear()
            continue

        # Accumulate enum value statements that may span multiple lines.
        if not stmt_buf:
            # A naive but practical heuristic: enum value statements contain '='
            # and are not "option"/"reserved"/"extensions" directives.
            stripped = line.lstrip()
            if "=" not in line:
                continue
            if stripped.startswith("option ") or stripped.startswith(
                    "reserved ") or stripped.startswith("extensions "):
                continue
            stmt_buf.append(line)
        else:
            stmt_buf.append(line)

        if ";" not in line:
            continue

        stmt = " ".join(part.strip() for part in stmt_buf)
        stmt_buf.clear()

        mval = _RE_ENUM_VALUE.match(stmt)
        if not mval:
            continue

        value_name = mval.group("name")
        value_num = int(mval.group("num"))

        alias_match = _RE_XRES_ENUM_ALIAS.search(stmt)
        desc_match = _RE_ERROR_CODE_DESC.search(stmt)
        show_match = _RE_ERROR_CODE_SHOW.search(stmt)
        attr_mode_match = _RE_ATTRIBUTE_ENUM_MODE.search(stmt)

        info: dict[str, object] = {
            "name": value_name,
            "number": value_num,
        }
        if alias_match:
            info["org.xresloader.enum_alias"] = _unescape_proto_string(
                alias_match.group("alias"))
        if desc_match:
            info["error_code.description"] = _unescape_proto_string(
                desc_match.group("desc"))
        if show_match:
            info["error_code.show_code"] = (show_match.group("flag") == "true")
        if attr_mode_match:
            info["attribute_enum_extension.mode"] = attr_mode_match.group(
                "mode")

        # Only record values that actually have plugin info.
        if len(info.keys()) > 2:
            enums[cur_enum].append(info)

    # Drop empty enums
    return {k: v for k, v in enums.items() if v}


def _extract_message_field_plugin_info(
        proto_text: str) -> dict[str, list[dict[str, object]]]:
    """Extract per-field plugin metadata from .proto source.

    We currently care about:
      - (org.xresloader.field_alias) string on FieldOptions

    Note: We parse source text rather than descriptors to avoid requiring
    compiled Python descriptors for custom extensions.
    """

    messages: dict[str, list[dict[str, object]]] = {}

    depth = 0
    msg_stack: list[tuple[str, int]] = []  # (name, start_depth)
    stmt_buf: list[str] = []

    for raw_line in proto_text.splitlines():
        line = _remove_proto_line_comment(raw_line).rstrip()
        if not line.strip():
            continue

        # Detect message start at current depth (before applying brace delta).
        m = _RE_MESSAGE_START.match(line)
        if m:
            msg_stack.append((m.group("name"), depth))
            full_name = ".".join(n for n, _ in msg_stack)
            messages.setdefault(full_name, [])

        # If we are inside any message, try to accumulate a field statement.
        if msg_stack:
            if not stmt_buf:
                stripped = line.lstrip()
                if "=" not in line:
                    pass
                elif stripped.startswith("option ") or stripped.startswith(
                        "reserved ") or stripped.startswith("extensions "):
                    pass
                elif stripped.startswith("oneof ") or stripped.startswith(
                        "message ") or stripped.startswith(
                            "enum ") or stripped.startswith("service "):
                    pass
                else:
                    stmt_buf.append(line)
            else:
                stmt_buf.append(line)

            if stmt_buf and ";" in line:
                stmt = " ".join(part.strip() for part in stmt_buf)
                stmt_buf.clear()

                f = _RE_FIELD_DECL.search(stmt)
                if f:
                    field_name = f.group("name")
                    field_num = int(f.group("num"))

                    alias_match = _RE_XRES_FIELD_ALIAS.search(stmt)
                    if alias_match:
                        full_name = ".".join(n for n, _ in msg_stack)
                        messages.setdefault(full_name, []).append({
                            "name":
                            field_name,
                            "number":
                            field_num,
                            "org.xresloader.field_alias":
                            _unescape_proto_string(alias_match.group("alias")),
                        })

        # Apply brace delta after processing line content.
        depth += line.count("{")
        depth -= line.count("}")

        # Pop messages that ended on this line (may close multiple scopes).
        while msg_stack and depth <= msg_stack[-1][1]:
            msg_stack.pop()

        # If we left all messages, clear any partial statement buffer.
        if not msg_stack:
            stmt_buf.clear()

    # Drop empty messages
    return {k: v for k, v in messages.items() if v}


def _format_plugin_info_markdown(
    *,
    enum_info: dict[str, list[dict[str, object]]],
    field_info: dict[str, list[dict[str, object]]],
) -> str:
    if not enum_info and not field_info:
        return ""

    has_alias = any(
        any("org.xresloader.enum_alias" in it for it in items)
        for items in enum_info.values())
    has_field_alias = any(
        any("org.xresloader.field_alias" in it for it in items)
        for items in field_info.values())
    has_err_desc = any(
        any("error_code.description" in it for it in items)
        for items in enum_info.values())
    has_attr_mode = any(
        any("attribute_enum_extension.mode" in it for it in items)
        for items in enum_info.values())

    lines: list[str] = []
    lines.append("## 插件信息")
    lines.append("")
    lines.append("> 本节由脚本从 `.proto` 源码中的自定义 option 提取生成。")
    lines.append("")

    if has_alias or has_attr_mode:
        lines.append("### xresloader: enum_alias")
        if has_attr_mode:
            lines.append("")
            lines.append(
                "> Mode 列：`万分率` = 万分率模式（`EN_ATTRIBUTE_MODE_RATE`），`值` = 值模式")
        lines.append("")

        # Collect all enum names that have alias or attr_mode info.
        alias_enum_names = sorted({
            name
            for name, items in enum_info.items()
            if any("org.xresloader.enum_alias" in it
                   or "attribute_enum_extension.mode" in it for it in items)
        })
        for enum_name in alias_enum_names:
            # Merge: include values that have alias OR attr_mode.
            items = [
                it for it in enum_info[enum_name]
                if "org.xresloader.enum_alias" in it
                or "attribute_enum_extension.mode" in it
            ]
            if not items:
                continue

            # Deduplicate by value name in case both keys are on same value.
            seen: dict[str, dict[str, object]] = {}
            for it in items:
                name = str(it.get("name", ""))
                if name in seen:
                    seen[name].update(it)
                else:
                    seen[name] = dict(it)
            merged = list(seen.values())

            lines.append(f"#### {enum_name}")
            lines.append("")
            if has_attr_mode:
                lines.append("| Name | Number | Alias | Mode |")
                lines.append("| ---- | ------ | ----- | ---- |")
            else:
                lines.append("| Name | Number | Alias |")
                lines.append("| ---- | ------ | ----- |")
            for it in sorted(merged, key=lambda x: int(x.get("number", 0))):
                alias = it.get("org.xresloader.enum_alias", "")
                if has_attr_mode:
                    raw_mode = str(it.get("attribute_enum_extension.mode", ""))
                    if raw_mode == "EN_ATTRIBUTE_MODE_RATE":
                        mode_text = "万分率"
                    else:
                        mode_text = "值"
                    lines.append(
                        f"| {it.get('name')} | {it.get('number')} | {alias} | {mode_text} |"
                    )
                else:
                    lines.append(
                        f"| {it.get('name')} | {it.get('number')} | {alias} |")
            lines.append("")

    if has_field_alias:
        lines.append("### xresloader: field_alias")
        lines.append("")
        for msg_name in sorted(field_info.keys()):
            items = [
                it for it in field_info[msg_name]
                if "org.xresloader.field_alias" in it
            ]
            if not items:
                continue
            lines.append(f"#### {msg_name}")
            lines.append("")
            lines.append("| Field | Number | Alias |")
            lines.append("| ----- | ------ | ----- |")
            for it in sorted(items, key=lambda x: int(x.get("number", 0))):
                lines.append(
                    f"| {it.get('name')} | {it.get('number')} | {it.get('org.xresloader.field_alias')} |"
                )
            lines.append("")

    if has_err_desc:
        lines.append("### error_code: description")
        lines.append("")
        for enum_name in sorted(enum_info.keys()):
            items = [
                it for it in enum_info[enum_name]
                if "error_code.description" in it
            ]
            if not items:
                continue
            lines.append(f"#### {enum_name}")
            lines.append("")
            lines.append("| Name | Number | Description | ShowCode |")
            lines.append("| ---- | ------ | ----------- | -------- |")
            for it in sorted(items, key=lambda x: int(x.get("number", 0))):
                show = it.get("error_code.show_code")
                show_text = "true" if show is True else (
                    "false" if show is False else "")
                desc = str(it.get("error_code.description", ""))
                desc = desc.replace("\n", "<br>")
                lines.append(
                    f"| {it.get('name')} | {it.get('number')} | {desc} | {show_text} |"
                )
            lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def append_plugin_info_to_generated_markdown(*, proto_file: Path,
                                             out_md: Path) -> None:
    """Append extracted plugin info to a generated markdown file (if any)."""

    if not out_md.is_file():
        warn(f"Generated markdown not found for {proto_file}: {out_md}")
        return

    try:
        proto_text = proto_file.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        proto_text = proto_file.read_text(encoding="utf-8", errors="replace")

    enum_info = _extract_enum_value_plugin_info(proto_text)
    field_info = _extract_message_field_plugin_info(proto_text)
    extra = _format_plugin_info_markdown(enum_info=enum_info,
                                         field_info=field_info)
    if not extra:
        return

    md = out_md.read_text(encoding="utf-8")
    md = md.rstrip() + "\n\n---\n\n" + extra
    out_md.write_text(md, encoding="utf-8")


def title_from_filename(name: str) -> str:
    # Prefer the proto base name as the display text.
    # Example: lobbysvr.com.protocol.user.md -> lobbysvr.com.protocol.user
    if name.lower().endswith(".md"):
        return name[:-3]
    return name


def write_protocols_index(out_root: Path) -> None:
    """Create an index page under protocols/ that links to all generated pages.

    We group by directory (relative to out_root) to keep it readable and to
    preserve the original protocol directory structure.
    """

    rel_files: list[Path] = []
    for p in out_root.rglob("*.md"):
        if not p.is_file():
            continue
        if p.name.lower() == "index.md":
            continue
        rel_files.append(p.relative_to(out_root))

    rel_files.sort(key=lambda x: x.as_posix())

    # Group by parent directory
    grouped: dict[str, list[Path]] = {}
    for rel in rel_files:
        key = rel.parent.as_posix() if rel.parent.as_posix(
        ) != "." else "(root)"
        grouped.setdefault(key, []).append(rel)

    lines: list[str] = []
    lines.append("# 协议文档")
    lines.append("")
    lines.append("> 本页由脚本自动生成，请勿手工编辑。")
    lines.append(
        "> 来源：`src\\lobbysvr\\protocol\\public` 与 `src\\component\\protocol\\public` 下所有 `.proto`。"
    )
    lines.append("")

    # Quick links to top-level directories
    top_dirs = sorted([p for p in out_root.iterdir() if p.is_dir()],
                      key=lambda p: p.name)
    if top_dirs:
        lines.append("## 快速入口")
        lines.append("")
        for d in top_dirs:
            # Anchor uses the heading generated below (directory headings).
            # We use a direct link to the first group section by directory name.
            lines.append(f"- `{d.name}/`")
        lines.append("")

    lines.append("## 目录")
    lines.append("")

    for dir_key in sorted(grouped.keys()):
        if dir_key == "(root)":
            lines.append("### protocols/")
        else:
            lines.append(f"### {dir_key}/")
        lines.append("")
        for rel in grouped[dir_key]:
            display = title_from_filename(rel.name)
            href = rel.as_posix()
            lines.append(f"- [{display}]({href})")
        lines.append("")

    (out_root / "index.md").write_text("\n".join(lines), encoding="utf-8")


def log(msg: str) -> None:
    print(f"[proto-docs] {msg}")


def warn(msg: str) -> None:
    print(f"[proto-docs] WARNING: {msg}", file=sys.stderr)


def err(msg: str) -> None:
    print(f"[proto-docs] ERROR: {msg}", file=sys.stderr)


def repo_root_from_doc_dir(doc_dir: Path) -> Path:
    return doc_dir.parent


def read_build_settings_protoc(build_settings_file: Path) -> Path | None:
    if not build_settings_file.is_file():
        return None
    try:
        data = json.loads(build_settings_file.read_text(encoding="utf-8"))
        p = data.get("tools", {}).get("protoc", {}).get("path")
        if not p:
            return None
        protoc_path = Path(p)
        if protoc_path.exists():
            return protoc_path
        return None
    except Exception as e:
        warn(f"Failed to parse {build_settings_file}: {e}")
        return None


def which(exe_name: str) -> str | None:
    return shutil.which(exe_name)


def ensure_protoc_gen_doc(repo_root: Path) -> Path:
    """Ensure protoc-gen-doc is available; install into tools/bin if needed."""

    tools_bin = repo_root / "tools" / "bin"
    candidate = tools_bin / ("protoc-gen-doc.exe" if sys.platform.startswith(
        "win") else "protoc-gen-doc")
    if candidate.exists():
        return candidate

    existing = which("protoc-gen-doc")
    if existing:
        return Path(existing)

    go = which("go")
    if not go:
        raise RuntimeError(
            "protoc-gen-doc not found on PATH, and Go is not available to install it."
        )

    tools_bin.mkdir(parents=True, exist_ok=True)

    env = os.environ.copy()
    env["GOBIN"] = str(tools_bin)

    def _try_go_install(install_env: dict[str, str], note: str) -> bool:
        try:
            log(f"Installing protoc-gen-doc into {tools_bin} ({note}) ...")
            subprocess.run(
                [
                    go, "install",
                    "github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest"
                ],
                check=True,
                env=install_env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )
            return True
        except subprocess.CalledProcessError as e:
            # Common CI issue: sum.golang.org unavailable (503) or blocked.
            stderr = (e.stderr or "").strip()
            if stderr:
                warn(f"go install failed ({note}): {stderr}")
            else:
                warn(f"go install failed ({note}): {e}")
            return False

    # First attempt: inherit user settings.
    if not _try_go_install(env, "default"):
        # Second attempt: bypass checksum DB (common when sum.golang.org is unreachable).
        env_no_sumdb = env.copy()
        env_no_sumdb["GOSUMDB"] = "off"
        # Keep GOPROXY if user set it; otherwise fall back to direct.
        env_no_sumdb.setdefault("GOPROXY", "direct")
        if not _try_go_install(env_no_sumdb, "GOSUMDB=off"):
            # Third attempt: try with a proxy that may work better in restricted networks.
            env_proxy = env.copy()
            env_proxy["GOSUMDB"] = "off"
            env_proxy["GOPROXY"] = env.get(
                "GOPROXY") or "https://goproxy.cn,direct"
            if not _try_go_install(
                    env_proxy, f"GOPROXY={env_proxy['GOPROXY']},GOSUMDB=off"):
                raise RuntimeError(
                    "Failed to install protoc-gen-doc via go install. "
                    "On CI/Linux this is often caused by sum.golang.org being unreachable. "
                    "Try setting environment variables like GOPROXY=https://goproxy.cn,direct and/or GOSUMDB=off."
                )

    # Make it visible for subsequent protoc invocations.
    env_path = env.get("PATH", "")
    env["PATH"] = str(tools_bin) + os.pathsep + env_path

    installed = tools_bin / ("protoc-gen-doc.exe" if sys.platform.startswith(
        "win") else "protoc-gen-doc")
    if installed.exists():
        return installed

    # Fallback to which() after installation.
    existing = shutil.which("protoc-gen-doc", path=env["PATH"])
    if existing:
        return Path(existing)

    raise RuntimeError("Failed to install protoc-gen-doc")


def resolve_protoc(repo_root: Path) -> Path:
    build_settings = repo_root / "build" / "build-settings.json"
    protoc = read_build_settings_protoc(build_settings)
    if protoc:
        return protoc

    existing = which("protoc")
    if existing:
        return Path(existing)

    raise RuntimeError(
        "protoc not found. Configure it in build/build-settings.json or install protoc and add it to PATH."
    )


def iter_proto_files(root: Path) -> list[Path]:
    return sorted([p for p in root.rglob("*.proto") if p.is_file()])


def compute_output_rel_md(*, proto_root: Path, proto_file: Path,
                          strip_first_dir: bool) -> Path:
    """Compute the output .md path relative to the docs output root."""

    rel = proto_file.relative_to(proto_root)
    if strip_first_dir:
        parts = rel.parts
        if len(parts) >= 2:
            rel = Path(*parts[1:])
    return rel.with_suffix(".md")


def run_protoc_doc(
    *,
    protoc: Path,
    env: dict[str, str],
    includes: list[Path],
    out_root: Path,
    proto_root: Path,
    proto_file: Path,
    strip_first_dir: bool,
) -> None:
    out_rel = compute_output_rel_md(
        proto_root=proto_root,
        proto_file=proto_file,
        strip_first_dir=strip_first_dir,
    )
    out_dir = out_root / out_rel.parent
    out_dir.mkdir(parents=True, exist_ok=True)

    args: list[str] = [str(protoc)]
    for inc in includes:
        args.extend(["-I", str(inc)])

    # protoc-gen-doc's --doc_opt takes a filename (not reliably a nested path).
    # To preserve directory structure, we set --doc_out to the target directory
    # for each proto and use a plain filename for --doc_opt.
    args.extend([
        f"--doc_out={out_dir}",
        f"--doc_opt=markdown,{out_rel.name}",
        str(proto_file),
    ])

    subprocess.run(args, check=True, env=env)


def main() -> int:
    doc_dir = Path(__file__).resolve().parent.parent
    repo_root = repo_root_from_doc_dir(doc_dir)

    lobbysvr_root = repo_root / "src" / "lobbysvr" / "protocol" / "public"
    component_root = repo_root / "src" / "component" / "protocol" / "public"

    xres_core_root = repo_root / "third_party" / "xresloader" / "protocols" / "core"
    xres_code_root = repo_root / "third_party" / "xresloader" / "protocols" / "code"

    out_root = doc_dir / "docs" / "protocols"

    for d in (lobbysvr_root, component_root, xres_core_root, xres_code_root):
        if not d.is_dir():
            raise RuntimeError(f"Directory not found: {d}")

    protoc = resolve_protoc(repo_root)
    _ = ensure_protoc_gen_doc(repo_root)

    # Ensure protoc can find protoc-gen-doc via PATH.
    env = os.environ.copy()
    tools_bin = repo_root / "tools" / "bin"
    env["PATH"] = str(tools_bin) + os.pathsep + env.get("PATH", "")

    includes = [
        lobbysvr_root,
        component_root / "common",
        component_root / "pbdesc",
        component_root / "extension",
        component_root / "config",
        xres_core_root,
        xres_code_root,
    ]

    log(f"Using protoc: {protoc}")
    log(f"Output: {out_root}")

    if out_root.exists():
        shutil.rmtree(out_root)
    out_root.mkdir(parents=True, exist_ok=True)

    for proto_root in (lobbysvr_root, component_root):
        strip_first_dir = (proto_root == component_root)
        protos = iter_proto_files(proto_root)
        log(f"Scanning {proto_root} ({len(protos)} proto files)")
        for p in protos:
            run_protoc_doc(
                protoc=protoc,
                env=env,
                includes=includes,
                out_root=out_root,
                proto_root=proto_root,
                proto_file=p,
                strip_first_dir=strip_first_dir,
            )

            out_md = out_root / compute_output_rel_md(
                proto_root=proto_root,
                proto_file=p,
                strip_first_dir=strip_first_dir,
            )
            append_plugin_info_to_generated_markdown(proto_file=p,
                                                     out_md=out_md)
            remove_scalar_value_types_section(out_md)
            normalize_markdown_tables(out_md)

    write_protocols_index(out_root)

    log("Done")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except subprocess.CalledProcessError as e:
        err(f"Command failed: {e}")
        raise
    except Exception as e:
        err(str(e))
        raise
