import os
import sqlite3

BASE_DIR = "/home/maeda/Documents/projects/goQuant/data/enhanced_wsdata"

def inspect_sqlite_db(db_path):
    print(f"\n=== Inspecting {db_path} ===")
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()

    # 查询所有表
    cur.execute("SELECT name FROM sqlite_master WHERE type='table';")
    tables = [row[0] for row in cur.fetchall()]

    if not tables:
        print("No tables found.")
        return

    for table in tables:
        print(f"\n--- Table: {table} ---")

        # 打印 schema
        cur.execute(f"PRAGMA table_info({table});")
        schema = cur.fetchall()
        print("Schema:")
        for col in schema:
            cid, name, dtype, notnull, dflt, pk = col
            print(f"  {name:<15} {dtype}")

        # 取前几行看看内容
        cur.execute(f"SELECT * FROM {table} LIMIT 10;")
        rows = cur.fetchall()

        print("\nSample Rows:")
        if rows:
            for r in rows:
                print(" ", r)
        else:
            print("  (empty)")

    conn.close()
    print("====================================")


def main():
    print(f"Scanning directory: {BASE_DIR}\n")

    for root, dirs, files in os.walk(BASE_DIR):
        for f in files:
            if f.endswith(".db") or f.endswith(".sqlite"):
                inspect_sqlite_db(os.path.join(root, f))


if __name__ == "__main__":
    main()
