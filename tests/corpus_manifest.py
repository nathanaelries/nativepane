"""Offline provenance and claim coverage gate. No baseline rewriting or network."""
import fnmatch
import hashlib
import json
import re
import xml.etree.ElementTree as E
from pathlib import Path
from zipfile import ZipFile

NS = {
    'w': 'http://schemas.openxmlformats.org/wordprocessingml/2006/main',
    's': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main',
    'a': 'http://schemas.openxmlformats.org/drawingml/2006/main',
    'p': 'http://schemas.openxmlformats.org/presentationml/2006/main',
    'm': 'http://schemas.openxmlformats.org/officeDocument/2006/math',
}
ROOT = Path(__file__).resolve().parent.parent
FIXTURES = ROOT / 'tests/fixtures'


def validate(manifest, require_visual=False):
    assert manifest['version'] == 2
    sources = {(s['repository'], s['revision']): s for s in manifest['sources']}
    assert len(sources) == len(manifest['sources']), 'duplicate source revision'
    for source in sources.values():
        assert hashlib.sha256((FIXTURES / source['licenseFile']).read_bytes()).hexdigest() == source['licenseSha256']
        if source.get('noticeFile'):
            assert (FIXTURES / source['noticeFile']).is_file()
    claims = {c['id']: c for c in manifest['layoutClaims']}
    assert len(claims) == len(manifest['layoutClaims']), 'duplicate claim ID'
    for claim in claims.values():
        assert claim['status'] in ('approximate', 'planned', 'preserved')
        assert claim['format'] in ('docx', 'xlsx', 'pptx')
        assert (ROOT / claim['reference'].split('#')[0]).is_file()
    covered = set()
    ids, files, baseline_paths = set(), set(), set()
    problems = []
    for fixture in manifest['fixtures']:
        try:
            name = fixture['id']
            assert re.fullmatch(r'[a-z0-9-]+', name) and name not in ids
            ids.add(name)
            assert fixture['file'] not in files
            files.add(fixture['file'])
            assert re.fullmatch(r'[0-9a-f]{40}', fixture['sourceRevision'])
            assert fixture['license'] in ('MIT', 'Apache-2.0')
            origin = sources[(fixture['sourceRepository'], fixture['sourceRevision'])]
            assert origin['license'] == fixture['license']
            source = FIXTURES / fixture['file']
            assert hashlib.sha256(source.read_bytes()).hexdigest() == fixture['sha256'], 'source hash changed'
            model = json.loads((FIXTURES / fixture['semantic']).read_text(encoding='utf-8'))
            assert model['format'] == fixture['format']
            assert sum(len(b['fields']) for b in model['blocks']) == fixture['projectedFields'], 'field inventory changed'
            assert model.get('readOnly', False) == fixture['readOnly']
            paths = [fixture['visual']] + [v['visual'] for v in fixture.get('views', [])]
            if fixture.get('layoutProbe'):
                paths.append(fixture['layoutProbe']['visual'])
            for path in paths:
                assert path.startswith('visual/') and path not in baseline_paths
                baseline_paths.add(path)
                if require_visual:
                    assert (FIXTURES / path).is_file(), 'missing visual baseline: ' + path
            with ZipFile(source) as package:
                props = E.fromstring(package.read('docProps/app.xml'))
                values = {e.tag.split('}')[-1]: e.text or '' for e in props}
                assert values.get('Application', '') == fixture['producer']
                assert values.get('AppVersion', '') == fixture['producerVersion']
                if not fixture['producer']:
                    assert fixture['producerEvidence'] and fixture['format'] == 'xlsx'
                    version = E.fromstring(package.read('xl/workbook.xml')).find('s:fileVersion', NS)
                    assert version is not None and version.attrib == {
                        'appName': 'xl', 'lastEdited': '4', 'lowestEdited': '4', 'rupBuild': '4505'}
                else:
                    assert fixture['producer'] in ('Microsoft Office Word', 'Microsoft Excel', 'Microsoft Office PowerPoint', 'Microsoft PowerPoint')
                for coverage in fixture['coverage']:
                    claim = coverage['claim']
                    assert claim in claims and claims[claim]['format'] == fixture['format']
                    assert coverage['evidence'], 'claim lacks source evidence'
                    for evidence in coverage['evidence']:
                        assert evidence['min'] > 0, 'zero evidence cannot establish coverage'
                        count = 0
                        parts = [n for n in package.namelist() if fnmatch.fnmatchcase(n, evidence['part'])]
                        for part in parts:
                            nodes = E.fromstring(package.read(part)).findall(evidence['path'], NS)
                            if evidence.get('tag'):
                                prefix, local = evidence['tag'].split(':')
                                nodes = [n for n in nodes if n.tag == '{' + NS[prefix] + '}' + local]
                            count += len(nodes)
                        assert count >= evidence['min'], f"{claim}: absent source construct {evidence} (found {count})"
                    covered.add(claim)
        except (AssertionError, KeyError, ValueError, OSError) as error:
            problems.append(fixture['id'] + ': ' + str(error))
    problems.extend('uncovered layout claim: ' + claim for claim in sorted(set(claims) - covered))
    assert not problems, '\n'.join(problems)
    return len(ids), len(claims), len(baseline_paths)


if __name__ == '__main__':
    result = validate(json.loads((FIXTURES / 'manifest.json').read_text(encoding='utf-8')))
    print('Claim/provenance gate PASS: %d Office fixtures, %d layout claims, %d visual views' % result)
