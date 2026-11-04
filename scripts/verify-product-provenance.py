import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import zipfile


def git(root, *arguments):
    return subprocess.check_output(['git', '-C', str(root), *arguments], stderr=subprocess.PIPE)


def object_id(data, algorithm):
    return hashlib.new(algorithm, b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()


def normalized(data):
    try:
        value = re.sub(r'\s+', '', data.decode('utf-8'))
    except UnicodeDecodeError:
        return None
    if len(value) < 80:
        return None
    return hashlib.sha256(value.encode()).hexdigest()


def candidate(root, path, data, candidates):
    key = str(path)
    candidates[key] = {'bytes': len(data), 'object': object_id(data, 'sha1'), 'normalized': normalized(data), 'sha256': hashlib.sha256(data).hexdigest()}
    if key.endswith('.zip'):
        with zipfile.ZipFile(root / path) as archive:
            if sum(entry.file_size for entry in archive.infolist()) > 128 * 1024 * 1024:
                raise ValueError('Tracked ZIP exceeds the provenance audit limit: ' + key)
            for entry in archive.infolist():
                if not entry.is_dir():
                    body = archive.read(entry)
                    candidates[key + '!' + entry.filename] = {'bytes': len(body), 'object': object_id(body, 'sha1'), 'normalized': normalized(body), 'sha256': hashlib.sha256(body).hexdigest()}


def main():
    parser = argparse.ArgumentParser(description='Compare tracked product material and ZIP members with explicit read-only reference checkouts.')
    parser.add_argument('--reference-repository', type=Path, action='append', required=True)
    parser.add_argument('--output', type=Path, required=True)
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    creation = git(root, 'log', '--reverse', '--format=%cI').decode().splitlines()[0]
    candidates = {}
    paths = [Path(name.decode()) for name in git(root, 'ls-files', '-z').split(b'\0') if name]
    for path in paths:
        full = root / path
        if full.is_symlink():
            raise ValueError('Product provenance requires regular tracked files: ' + str(path))
        candidate(root, path, full.read_bytes(), candidates)
    hashes = {}
    normalized_hashes = {}
    for path, value in candidates.items():
        hashes.setdefault(value['object'], []).append(path)
        if value['normalized']:
            normalized_hashes.setdefault(value['normalized'], []).append(path)
    report = {'schemaVersion': 1, 'productCommit': git(root, 'rev-parse', 'HEAD').decode().strip(), 'includesCurrentTrackedWorktree': True, 'productCreation': creation, 'trackedFiles': len(paths), 'archiveMembers': len(candidates) - len(paths), 'referenceRepositories': [], 'matches': [], 'limitations': ['Content comparison flags exact and whitespace-normalized copies; it does not establish authorship by itself.', 'Exact comparison includes reference HEAD and its latest ancestor before product creation. Whitespace comparison covers tracked working files up to 8 MiB and normalized text of at least 80 characters.', 'Untracked reference material and arbitrary rewritten derivations are outside automated comparison. Review repository history and authored fixture provenance separately.']}
    seen = set()
    for source in arguments.reference_repository:
        source = source.resolve()
        if source == root or Path(git(source, 'rev-parse', '--show-toplevel').decode().strip()).resolve() != source:
            raise ValueError('Each reference must be a different Git repository root')
        if git(source, 'rev-parse', '--show-object-format').decode().strip() != 'sha1':
            raise ValueError('Reference object format is unsupported: ' + str(source))
        before = git(source, 'rev-list', '-1', '--before=' + creation, 'HEAD').decode().strip()
        versions = list(dict.fromkeys([git(source, 'rev-parse', 'HEAD').decode().strip(), before]))
        reference = {'root': str(source), 'snapshots': [ref for ref in versions if ref], 'trackedWorkingFiles': 0, 'largeWorkingFiles': [], 'missingWorkingFiles': [], 'symlinkWorkingFiles': []}
        report['referenceRepositories'].append(reference)
        for ref in reference['snapshots']:
            for item in git(source, 'ls-tree', '-rz', ref).split(b'\0'):
                if not item:
                    continue
                metadata, path = item.split(b'\t', 1)
                _, kind, oid = metadata.decode().split()
                if kind != 'blob':
                    continue
                for product in hashes.get(oid, []):
                    key = (product, str(source), path.decode(), 'exact')
                    if key not in seen:
                        seen.add(key)
                        report['matches'].append({'product': product, 'referenceRepository': str(source), 'referencePath': path.decode(), 'kind': 'exact', 'snapshot': ref})
        for item in git(source, 'ls-files', '-z').split(b'\0'):
            if not item:
                continue
            path = source / item.decode()
            if path.is_symlink():
                reference['symlinkWorkingFiles'].append(item.decode())
                continue
            if not path.is_file():
                reference['missingWorkingFiles'].append(item.decode())
                continue
            reference['trackedWorkingFiles'] += 1
            if path.stat().st_size > 8 * 1024 * 1024:
                reference['largeWorkingFiles'].append(item.decode())
                continue
            data = path.read_bytes()
            oid = object_id(data, 'sha1')
            products = set(hashes.get(oid, []))
            digest = normalized(data)
            for product in products.union(normalized_hashes.get(digest, [])):
                method = 'exact' if product in products else 'whitespace-normalized'
                key = (product, str(source), item.decode(), method)
                if key not in seen:
                    seen.add(key)
                    report['matches'].append({'product': product, 'referenceRepository': str(source), 'referencePath': item.decode(), 'kind': method, 'snapshot': 'working-tree'})
    report['candidateInventory'] = candidates
    arguments.output.parent.mkdir(parents=True, exist_ok=True)
    arguments.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps({'trackedFiles': report['trackedFiles'], 'archiveMembers': report['archiveMembers'], 'referenceRepositories': len(report['referenceRepositories']), 'matches': len(report['matches']), 'report': str(arguments.output)}))
    return 1 if report['matches'] else 0


if __name__ == '__main__':
    raise SystemExit(main())
