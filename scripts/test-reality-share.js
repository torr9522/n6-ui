'use strict';

const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

global.ObjectUtil = {
    isEmpty(value) {
        return value === undefined || value === null || value === '';
    },
};
global.normalizeShareAddress = value => value;

const modelPath = path.join(__dirname, '..', 'web', 'assets', 'js', 'model', 'xray.js');
const modelSource = fs.readFileSync(modelPath, 'utf8') + '\n;globalThis.__N6Inbound = Inbound;';
vm.runInThisContext(modelSource, {filename: modelPath});

function createRealityInbound(serverNames) {
    return {
        port: 443,
        reality: true,
        xtls: false,
        settings: {
            decryption: 'none',
            vlesses: [{id: '11111111-1111-1111-1111-111111111111', flow: ''}],
        },
        stream: {
            network: 'tcp',
            security: 'reality',
            tcp: {type: 'none'},
            reality: {
                serverNames,
                shortIds: ['0123456789abcdef'],
                publicKey: 'TEST_PUBLIC_KEY',
                fingerprint: 'chrome',
                spiderX: '',
            },
        },
    };
}

function exportSNI(inbound, address) {
    const link = __N6Inbound.prototype.genVLESSLink.call(inbound, address, 'test');
    return {link, sni: new URL(link).searchParams.get('sni')};
}

const cases = [
    [['www.cloudflare.com'], 'www.cloudflare.com'],
    [['', 'www.cloudflare.com'], 'www.cloudflare.com'],
    [['', '', 'www.cloudflare.com'], 'www.cloudflare.com'],
    [[], '198.51.100.10'],
    [[''], '198.51.100.10'],
    [['   ', 'www.cloudflare.com'], 'www.cloudflare.com'],
];

for (const [serverNames, expected] of cases) {
    const inbound = createRealityInbound(serverNames);
    const before = JSON.stringify(inbound);
    assert.strictEqual(exportSNI(inbound, '198.51.100.10').sni, expected);
    assert.strictEqual(JSON.stringify(inbound), before, 'share export mutated the inbound object');
}

const historical = createRealityInbound(['', 'www.cloudflare.com']);
assert.strictEqual(exportSNI(historical, 'domain.test').sni, 'www.cloudflare.com');
assert.strictEqual(exportSNI(historical, '198.51.100.10').sni, 'www.cloudflare.com');

console.log('N6 Reality share frontend regression: PASS');
