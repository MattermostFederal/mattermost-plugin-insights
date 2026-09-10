import {expect, test} from '@playwright/test';

import {parseCSRFToken, requestHeaders} from './csrf';

test('reads the CSRF token out of the MMCSRF cookie', () => {
    expect(parseCSRFToken('MMCSRF=abc123')).toBe('abc123');
});

// document.cookie is a single string of every cookie on the domain; the host
// sets MMAUTHTOKEN and MMUSERID alongside it.
test('finds MMCSRF among the other host cookies', () => {
    const cookie = 'MMUSERID=xyz; MMCSRF=abc123; MMAUTHTOKEN=tok';
    expect(parseCSRFToken(cookie)).toBe('abc123');
});

// A cookie whose name merely ends in MMCSRF must not be mistaken for it.
test('does not match a cookie that only ends with the name', () => {
    expect(parseCSRFToken('NOTMMCSRF=nope')).toBe('');
});

test('returns empty when there is no CSRF cookie', () => {
    expect(parseCSRFToken('MMUSERID=xyz')).toBe('');
    expect(parseCSRFToken('')).toBe('');
});

// GET is exempt from the server's CSRF check (validateCSRFForPluginRequest
// returns early), so sending the token there would be noise.
test('omits the CSRF header on GET', () => {
    const headers = requestHeaders('GET', {}, () => 'abc123');
    expect(headers['X-CSRF-Token']).toBeUndefined();
});

test('sends the CSRF header on state-changing methods', () => {
    for (const method of ['POST', 'PUT', 'DELETE']) {
        const headers = requestHeaders(method, {}, () => 'abc123');
        expect(headers['X-CSRF-Token']).toBe('abc123');
    }
});

test('matches the method case-insensitively', () => {
    expect(requestHeaders('post', {}, () => 'abc123')['X-CSRF-Token']).toBe('abc123');
    expect(requestHeaders('get', {}, () => 'abc123')['X-CSRF-Token']).toBeUndefined();
});

// Without a readable cookie there is nothing to send; the legacy header is
// all that keeps the request working on a non-strict server.
test('omits the CSRF header when no token is available', () => {
    const headers = requestHeaders('POST', {}, () => '');
    expect(headers['X-CSRF-Token']).toBeUndefined();
    expect(headers['X-Requested-With']).toBe('XMLHttpRequest');
});

test('always sends the legacy X-Requested-With header', () => {
    expect(requestHeaders('GET', {}, () => 'abc123')['X-Requested-With']).toBe('XMLHttpRequest');
    expect(requestHeaders('POST', {}, () => 'abc123')['X-Requested-With']).toBe('XMLHttpRequest');
});

test('merges caller-supplied headers', () => {
    const headers = requestHeaders('POST', {'Content-Type': 'application/json'}, () => 'abc123');
    expect(headers['Content-Type']).toBe('application/json');
    expect(headers['X-CSRF-Token']).toBe('abc123');
});
