import '@angular/compiler'
import assert from 'node:assert/strict'
import { Component } from '@angular/core'
import { bootstrapApplication } from '@angular/platform-browser'
import { renderApplication } from '@angular/platform-server'
import { TestStudioStatusPageComponent, TestStudioStatusBarComponent } from '@wtiger001/teststudio-angular'

const row = { title: 'Server-rendered check', description: '', state: 'unknown', metric: 'outcomes', history: [], buckets: [], availability: { percent: null, coveragePercent: 0, upSeconds: 0, downSeconds: 0, unknownSeconds: 0, since: '2026-11-07T00:00:00Z' } }
class Consumer { report = { title: 'Server-rendered report', background: '', window: '24h', maxAgeSeconds: 0, asOf: '2026-11-07T00:00:00Z', probes: [row] }; row = row; source = { url: 'https://status.example.test/api/v1/public/status-pages/public' } }
Component({ selector: 'ssr-consumer', standalone: true, imports: [TestStudioStatusPageComponent, TestStudioStatusBarComponent], template: '<teststudio-status-page [report]="report"/><teststudio-status-bar [row]="row"/><teststudio-status-page [source]="source"/>' })(Consumer)
const html = await renderApplication(context => bootstrapApplication(Consumer, {}, context), { document: '<html><head></head><body><ssr-consumer></ssr-consumer></body></html>', url: 'https://consumer.example.test/', allowedHosts: ['consumer.example.test'] })
assert(html.includes('Server-rendered report'))
assert(html.includes('Server-rendered check'))
assert(html.includes('No observations in this period'))
console.log('Built-package page, bar, multiple instances, and public-source SSR pass without browser globals or HttpClient.')
