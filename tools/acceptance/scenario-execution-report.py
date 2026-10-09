"""Bind proposed T01-T96 test mappings to actual Go execution, not acceptance claims."""
import argparse
import datetime
import json
import re
import subprocess
from pathlib import Path


def execution_report(matrix, logs, source, root=Path('.')):
    if not re.fullmatch(r'[0-9a-f]{40}', source):
        raise ValueError('full immutable source required')
    rows = matrix['scenarios']
    if [r['id'] for r in rows] != [f'T{i:02}' for i in range(1, 97)]:
        raise ValueError('complete ordered T01-T96 scenario declaration required')
    module = re.search(r'^module (.+)$', (root/'go.mod').read_text(), re.M)[1]
    passed, skipped, packages = set(), set(), set()
    for log in logs:
        for line in log.read_text().splitlines():
            event = json.loads(line)
            action = event.get('Action')
            if action in {'fail', 'build-fail'}:
                raise ValueError('failed executions cannot produce passing evidence')
            package, name = event.get('Package', ''), event.get('Test')
            if action == 'pass' and not name:
                packages.add(package)
            if name and action in {'pass', 'skip'}:
                (passed if action == 'pass' else skipped).add((package, name))
    if not passed or not packages:
        raise ValueError('actual nonzero passing tests and packages required')
    mapped = []
    for row in rows:
        results = []
        for candidate in row['candidate_go_tests']:
            path, name = candidate['path'], candidate['test']
            file = root/path
            if not file.is_file() or not re.search(r'func\s+'+re.escape(name)+r'\s*\(', file.read_text()):
                raise ValueError(f"{row['id']} references a missing candidate function")
            package = module+'/'+str(Path(path).parent)
            actual_pass = (package, name) in passed and package in packages
            actual_skip = (package, name) in skipped
            results.append({'path': path, 'test': name, 'executed_pass': actual_pass,
                            'executed_skip': actual_skip,
                            'passing_subtests': sorted(n for p, n in passed if p == package and n.startswith(name+'/')),
                            'skipped_subtests': sorted(n for p, n in skipped if p == package and n.startswith(name+'/'))})
        mapped.append({'id': row['id'], 'required_assertion': row['required_assertion'],
                       'coverage_status': row['coverage_status'], 'candidate_test_results': results,
                       'other_resources': row['other_resources'], 'limitation': row['limitation'],
                       'assertion_audit_complete': False, 'scenario_accepted_automatically': False})
    return {'schema': 1, 'source_commit': source, 'recorded_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
            'scope': 'Actual test execution traceability. Matching test names do not prove all proposed assertions.',
            'go': subprocess.check_output(['go', 'version'], text=True).strip(),
            'execution_logs': [str(p) for p in logs], 'scenarios': mapped}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source')
    parser.add_argument('output', type=Path)
    parser.add_argument('logs', type=Path, nargs='+')
    args = parser.parse_args()
    matrix = json.loads(Path('docs/signed-acceptance-matrix.json').read_text())
    args.output.write_text(json.dumps(execution_report(matrix, args.logs, args.source), indent=2)+'\n')
    print('T01-T96 execution traceability recorded; no automatic scenario acceptance or coverage percentage')
