#!/usr/bin/env python3
import os
import json
import re
import sys

workspace = os.path.abspath(os.path.dirname(__file__))

CATEGORIES = {
    "ai": "AI/ML Related",
    "cka": "CKA (Certified Kubernetes Administrator) Related",
    "k8s": "Kubernetes CNCF Projects Related",
    "k8s-papers": "Kubernetes Related Papers",
    "general-knowledge-base": "General Knowledge Base",
    "system-design": "System Design",
    "how-facebook-xxx-series": "How Facebook Builds Systems",
    "how-google-xxx-series": "How Google Builds Systems",
    "how-amazon-xxx-series": "How Amazon Builds Systems",
    "how-uber-xxx-series": "How Uber Builds Systems",
    "how-alibaba-xxx-series": "How Alibaba Builds Systems",
}

def parse_title(file_path):
    ext = os.path.splitext(file_path)[1].lower()
    if ext == '.md':
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                for line in f:
                    line = line.strip()
                    if line.startswith('# '):
                        title = line[2:].strip()
                        if title.startswith('**') and title.endswith('**'):
                            title = title[2:-2].strip()
                        return title
        except Exception:
            pass
    elif ext == '.ipynb':
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                data = json.load(f)
                for cell in data.get('cells', []):
                    if cell.get('cell_type') == 'markdown':
                        for line in cell.get('source', []):
                            line = line.strip()
                            if line.startswith('# '):
                                title = line[2:].strip()
                                if title.startswith('**') and title.endswith('**'):
                                    title = title[2:-2].strip()
                                return title
        except Exception:
            pass
    return None

def humanize(name):
    parts = name.replace('-', ' ').replace('_', ' ').split()
    acronyms = {
        'ai': 'AI', 'ml': 'ML', 'rag': 'RAG', 'k8s': 'Kubernetes', 'cka': 'CKA',
        'oauth': 'OAuth', 'db': 'Database', 'vm': 'VM', 'saas': 'SaaS', 'csi': 'CSI',
        'gai': 'GAI', 'dra': 'DRA', 'tsdb': 'TSDB', 'uuid': 'UUID', 'rpc': 'RPC',
        'rest': 'REST', 'crdt': 'CRDT', 'sse': 'SSE', 'i18n': 'I18n'
    }
    capitalized = []
    for p in parts:
        lower_p = p.lower()
        if lower_p in acronyms:
            capitalized.append(acronyms[lower_p])
        else:
            capitalized.append(p.capitalize())
    return ' '.join(capitalized)

def get_note_info(rel_dir):
    full_dir = os.path.join(workspace, rel_dir)
    if not os.path.isdir(full_dir):
        return None
        
    files = []
    for entry in os.scandir(full_dir):
        if entry.is_file() and entry.name.lower().endswith(('.md', '.ipynb')):
            files.append(entry.name)
            
    if not files:
        return None
        
    primary = None
    for name in ['readme.md', 'README.md', 'notebook.ipynb']:
        if name in files:
            primary = name
            break
    if not primary:
        md_files = [f for f in files if f.lower().endswith('.md')]
        if md_files:
            primary = sorted(md_files)[0]
        else:
            primary = sorted(files)[0]
            
    secondary = sorted([f for f in files if f != primary])
    primary_path = os.path.join(full_dir, primary)
    title = parse_title(primary_path)
    if not title:
        title = humanize(os.path.basename(rel_dir))
        
    return {
        "dir": rel_dir,
        "primary_file": os.path.join(rel_dir, primary),
        "title": title,
        "secondary_files": [os.path.join(rel_dir, f) for f in secondary]
    }

def process_category(cat_name):
    cat_dir = os.path.join(workspace, cat_name)
    if not os.path.exists(cat_dir):
        return []
        
    items = []
    
    if cat_name == "system-design":
        chapters = []
        topics = []
        general = []
        
        for entry in os.scandir(cat_dir):
            if entry.is_dir():
                if entry.name.startswith(('.', '_')):
                    continue
                if entry.name == "topics":
                    topics_dir = os.path.join(cat_dir, "topics")
                    for sub in os.scandir(topics_dir):
                        if sub.is_dir() and not sub.name.startswith('.'):
                            note = get_note_info(os.path.join("system-design", "topics", sub.name))
                            if note:
                                topics.append(note)
                        elif sub.is_file() and sub.name.lower().endswith(('.md', '.ipynb')):
                            sub_title = parse_title(sub.path) or humanize(os.path.splitext(sub.name)[0])
                            topics.append({
                                "dir": "system-design/topics",
                                "primary_file": os.path.join("system-design", "topics", sub.name),
                                "title": sub_title,
                                "secondary_files": []
                            })
                elif re.match(r'^\d+', entry.name):
                    note = get_note_info(os.path.join("system-design", entry.name))
                    if note:
                        chapters.append(note)
                else:
                    note = get_note_info(os.path.join("system-design", entry.name))
                    if note:
                        general.append(note)
                        
        chapters.sort(key=lambda x: int(re.match(r'^\d+', os.path.basename(x['dir'])).group()))
        topics.sort(key=lambda x: x['title'].lower())
        general.sort(key=lambda x: x['title'].lower())
        
        return {
            "type": "system-design",
            "chapters": chapters,
            "topics": topics,
            "general": general
        }
        
    if cat_name == "cka":
        direct_files = []
        subdirs = []
        for entry in os.scandir(cat_dir):
            if entry.is_file() and entry.name.lower().endswith(('.md', '.ipynb')):
                title = parse_title(entry.path) or humanize(os.path.splitext(entry.name)[0])
                direct_files.append({
                    "primary_file": os.path.join("cka", entry.name),
                    "title": title,
                    "secondary_files": []
                })
            elif entry.is_dir() and not entry.name.startswith('.'):
                note = get_note_info(os.path.join("cka", entry.name))
                if note:
                    subdirs.append(note)
        
        direct_files.sort(key=lambda x: x['title'].lower())
        subdirs.sort(key=lambda x: x['title'].lower())
        return {
            "type": "cka",
            "direct_files": direct_files,
            "subdirs": subdirs
        }

    for entry in os.scandir(cat_dir):
        if entry.is_dir():
            if entry.name.startswith(('.', '_')):
                continue
            note = get_note_info(os.path.join(cat_name, entry.name))
            if note:
                nested_dir_path = os.path.join(workspace, cat_name, entry.name)
                nested_notes = []
                for sub_entry in os.scandir(nested_dir_path):
                    if sub_entry.is_dir() and not sub_entry.name.startswith(('.', '_', 'resources', 'resource')):
                        nested_note = get_note_info(os.path.join(cat_name, entry.name, sub_entry.name))
                        if nested_note:
                            nested_notes.append(nested_note)
                if nested_notes:
                    nested_notes.sort(key=lambda x: x['title'].lower())
                    note['nested_notes'] = nested_notes
                items.append(note)
        elif entry.is_file() and entry.name.lower().endswith(('.md', '.ipynb')) and entry.name.lower() not in ['readme.md', 'license']:
            title = parse_title(entry.path) or humanize(os.path.splitext(entry.name)[0])
            items.append({
                "dir": cat_name,
                "primary_file": os.path.join(cat_name, entry.name),
                "title": title,
                "secondary_files": []
            })
            
    items.sort(key=lambda x: x['title'].lower())
    return {
        "type": "regular",
        "items": items
    }

def generate_toc_markdown():
    lines = []
    for cat_key, cat_title in CATEGORIES.items():
        lines.append(f"### {cat_title}\n\n")
        data = process_category(cat_key)
        if not data:
            lines.append("*(No notes found)*\n\n")
            continue
            
        if data["type"] == "system-design":
            if data["general"]:
                lines.append("- Architecture & Patterns\n")
                for item in data["general"]:
                    lines.append(f"  - [{item['title']}]({item['primary_file']})\n")
            if data["chapters"]:
                lines.append("- Data-Intensive Applications (Book Notes)\n")
                for item in data["chapters"]:
                    lines.append(f"  - [{item['title']}]({item['primary_file']})\n")
                    for sec in item['secondary_files']:
                        sec_title = parse_title(os.path.join(workspace, sec)) or humanize(os.path.splitext(os.path.basename(sec))[0])
                        lines.append(f"    - [{sec_title}]({sec})\n")
            if data["topics"]:
                lines.append("- System Design Case Studies\n")
                for item in data["topics"]:
                    lines.append(f"  - [{item['title']}]({item['primary_file']})\n")
                    for sec in item['secondary_files']:
                        sec_title = parse_title(os.path.join(workspace, sec)) or humanize(os.path.splitext(os.path.basename(sec))[0])
                        lines.append(f"    - [{sec_title}]({sec})\n")
                        
        elif data["type"] == "cka":
            for item in data["direct_files"]:
                lines.append(f"- [{item['title']}]({item['primary_file']})\n")
            for item in data["subdirs"]:
                lines.append(f"- [{item['title']}]({item['primary_file']})\n")
                for sec in item['secondary_files']:
                    sec_title = parse_title(os.path.join(workspace, sec)) or humanize(os.path.splitext(os.path.basename(sec))[0])
                    lines.append(f"  - [{sec_title}]({sec})\n")
                    
        else:
            for item in data["items"]:
                lines.append(f"- [{item['title']}]({item['primary_file']})\n")
                for sec in item.get('secondary_files', []):
                    sec_title = parse_title(os.path.join(workspace, sec)) or humanize(os.path.splitext(os.path.basename(sec))[0])
                    lines.append(f"  - [{sec_title}]({sec})\n")
                for nest in item.get('nested_notes', []):
                    lines.append(f"  - [{nest['title']}]({nest['primary_file']})\n")
                    for sec in nest.get('secondary_files', []):
                        sec_title = parse_title(os.path.join(workspace, sec)) or humanize(os.path.splitext(os.path.basename(sec))[0])
                        lines.append(f"    - [{sec_title}]({sec})\n")
        lines.append("\n")
    return "".join(lines)

def main():
    dry_run = "--dry-run" in sys.argv
    
    readme_path = os.path.join(workspace, "README.md")
    if not os.path.exists(readme_path):
        header = "# Tech Notes\n\n## Table of contents\n\n"
    else:
        with open(readme_path, 'r', encoding='utf-8') as f:
            current_content = f.read()
        toc_marker = "## Table of contents"
        if toc_marker in current_content:
            header = current_content.split(toc_marker)[0] + toc_marker + "\n\n"
        else:
            header = current_content + "\n\n## Table of contents\n\n"
            
    toc_content = generate_toc_markdown()
    new_content = header + toc_content
    
    if dry_run:
        print(new_content)
    else:
        with open(readme_path, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print("Successfully updated README.md Table of Contents!")

if __name__ == "__main__":
    main()
