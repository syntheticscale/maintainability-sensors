import json
import glob
import sys
import os

def parse_chat_dumps():
    dump_dir = "chat_dumps"
    files = glob.glob(os.path.join(dump_dir, "*.jsonl"))
    
    if not files:
        print("No jsonl files found in chat_dumps/")
        return

    print("## Extracted Sessions Summary\n")
    
    for file_path in sorted(files):
        print(f"### File: {os.path.basename(file_path)}")
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                user_prompts = []
                agent_tasks = []
                for line in f:
                    if not line.strip(): continue
                    try:
                        entry = json.loads(line)
                        if entry.get("type") == "user":
                            content = entry.get("content", [])
                            for c in content:
                                if "text" in c:
                                    # limit length for display
                                    text = c["text"]
                                    # clean up giant prompts potentially
                                    if len(text) > 500:
                                        text = text[:500] + " ... [TRUNCATED]"
                                    user_prompts.append(text.strip())
                        elif entry.get("type") == "gemini":
                            thoughts = entry.get("thoughts", [])
                            for t in thoughts:
                                if "description" in t:
                                    agent_tasks.append(t["description"])
                    except json.JSONDecodeError:
                        continue
                        
                print("#### User Prompts:")
                for i, p in enumerate(user_prompts, 1):
                    print(f"{i}. {p}")
                print("\n#### Agent Thoughts (Tasks):")
                # Deduplicate or just show first few
                for i, t in enumerate(agent_tasks[:10], 1):
                     print(f"- {t}")
                if len(agent_tasks) > 10:
                     print(f"- ... and {len(agent_tasks)-10} more thoughts.")
                print("\n" + "-"*40 + "\n")
        except Exception as e:
            print(f"Error reading {file_path}: {e}")

if __name__ == '__main__':
    parse_chat_dumps()
