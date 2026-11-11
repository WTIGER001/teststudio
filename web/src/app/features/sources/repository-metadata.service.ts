import { inject, Injectable, signal } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AccessClientService } from '../../core/access-client.service';
import type { SourceRepositoryMetadata, SourceRepositoryMetadataRecord, SourceRepositoryMetadataPage, SourceRepositoryRecord, SourceRepositoryWrite } from '../../core/generated/access.types';
import { randomId } from '../../core/random-id';
@Injectable()
export class RepositoryMetadataService {
    private readonly client = inject(AccessClientService);
    private readonly route = inject(ActivatedRoute);
    private readonly router = inject(Router);
    readonly project = this.route.snapshot.paramMap.get('projectId')!;
    readonly source = this.route.snapshot.paramMap.get('sourceId')!;
    readonly suite = this.route.snapshot.paramMap.get('suiteId')!;
    readonly connection = signal<SourceRepositoryRecord | null>(null);
    readonly saved = signal<SourceRepositoryMetadataRecord | null>(null);
    readonly entries = signal<Record<string, unknown>[]>([]);
    readonly editingTest = signal<number | null>(null);
    readonly testForm = new FormGroup({ title: new FormControl('', { nonNullable: true }), description: new FormControl('', { nonNullable: true }), category: new FormControl('', { nonNullable: true }), owner: new FormControl('', { nonNullable: true }), safety: new FormControl('needs-review', { nonNullable: true }), rationale: new FormControl('', { nonNullable: true }), conditions: new FormControl('', { nonNullable: true }), parameters: new FormControl('', { nonNullable: true }), bindings: new FormControl('', { nonNullable: true }), enabled: new FormControl(false, { nonNullable: true }), reviewed: new FormControl(false, { nonNullable: true }) });
    readonly text = new FormControl('', { nonNullable: true });
    readonly busy = signal(false);
    readonly error = signal('');
    readonly message = signal('');
    readonly result = signal<SourceRepositoryWrite | null>(null);
    readonly review = signal(false);
    readonly history = signal<SourceRepositoryMetadataPage>({ items: [] });
    readonly remote = signal<SourceRepositoryMetadata | null>(null);
    readonly reviewedBase = signal<SourceRepositoryMetadata['base'] | undefined>(undefined);
    constructor() { this.text.valueChanges.subscribe(value => { try {
        const document = JSON.parse(value);
        this.entries.set(Array.isArray(document.catalog?.tests) ? document.catalog.tests : []);
    }
    catch {
        this.entries.set([]);
    } }); void this.load(); }
    editTest(index: number): void { const test = this.entries()[index]; this.editingTest.set(index); this.testForm.reset({ title: String(test['title'] ?? ''), description: String(test['description'] ?? ''), category: String(test['category'] ?? ''), owner: String(test['owner'] ?? ''), safety: String(test['safety'] ?? 'needs-review'), rationale: String(test['rationale'] ?? ''), conditions: String(test['conditions'] ?? ''), parameters: Array.isArray(test['requiredParameters']) ? test['requiredParameters'].join(', ') : '', bindings: Array.isArray(test['requiredBindings']) ? test['requiredBindings'].join(', ') : '', enabled: test['state'] === 'enabled', reviewed: false }); this.testForm.markAsPristine(); }
    applyTest(): void { const index = this.editingTest(); if (index === null)
        return; const value = this.testForm.getRawValue(); if (!value.title.trim() || !value.category.trim() || !value.rationale.trim() || value.enabled && !value.reviewed) {
        this.error.set('Provide a title, category, and safety rationale. Confirm review before enabling a test.');
        return;
    } ; const names = (text: string) => text.split(',').map(item => item.trim()).filter(Boolean); const document = JSON.parse(this.text.value); const before = document.catalog.tests[index]; document.catalog.tests[index] = { ...before, title: value.title.trim(), description: value.description, category: value.category.trim(), owner: value.owner, safety: value.safety, rationale: value.rationale, conditions: value.conditions, requiredParameters: names(value.parameters), requiredBindings: names(value.bindings), state: value.enabled ? 'enabled' : 'disabled' }; this.text.setValue(JSON.stringify(document, null, 2)); this.text.markAsDirty(); this.review.set(false); this.editingTest.set(null); this.testForm.markAsPristine(); this.error.set(''); }
    cancelTest(): void { this.editingTest.set(null); this.testForm.markAsPristine(); }
    private scope() { return { project: this.project, source: this.source, suite: this.suite }; }
    async load(): Promise<void> { if (this.text.dirty && !window.confirm('Reload and replace unsaved metadata edits?'))
        return; this.busy.set(true); this.error.set(''); try {
        const [connection, metadata] = await Promise.all([this.client.call('readRepositorySource', this.scope()), this.client.call('readRepositoryMetadata', this.scope())]);
        this.connection.set(connection);
        this.saved.set(metadata);
        this.text.setValue(JSON.stringify({ manifest: metadata.data.manifest, catalog: metadata.data.catalog }, null, 2));
        this.text.markAsPristine();
        this.review.set(false);
        this.reviewedBase.set(undefined);
        this.remote.set(null);
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message + ' Synchronize and finish source setup before opening metadata.' : 'Metadata could not be loaded.');
    }
    finally {
        this.busy.set(false);
    } }
    preview(): void { try {
        this.value();
        this.review.set(true);
        this.error.set('');
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Metadata JSON is invalid.');
    } }
    value(): SourceRepositoryMetadata { const value = JSON.parse(this.text.value); if (!value.manifest || !value.catalog)
        throw new Error('Provide a manifest and catalog object.'); return { ...this.saved()!.data, manifest: value.manifest, catalog: value.catalog, base: this.reviewedBase() ?? this.saved()!.data.base }; }
    async save(): Promise<void> { if (!this.review() || !this.saved() || this.busy())
        return; this.busy.set(true); this.error.set(''); try {
        const saved = await this.client.call('saveRepositoryMetadata', { ...this.scope(), revision: this.saved()!.revision, body: this.value() });
        this.saved.set(saved);
        this.text.markAsPristine();
        this.review.set(false);
        this.result.set(null);
        this.message.set(this.connection()?.data.configuration.accessMode === 'read-write' ? 'Pending metadata saved. Commit a review branch to send these changes to Git.' : 'Metadata saved in TestStudio. Synchronization will prepare it with the selected source commit. No tests started.');
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Metadata save failed. Edits are preserved.');
    }
    finally {
        this.busy.set(false);
    } }
    async commit(): Promise<void> { if (!this.saved() || this.text.dirty || this.busy())
        return; this.busy.set(true); this.error.set(''); try {
        const result = await this.client.call('commitRepositoryMetadata', { ...this.scope(), revision: this.saved()!.revision, key: 'metadata-' + this.saved()!.revision });
        this.result.set(result);
        this.message.set(result.state === 'committed' ? 'Metadata committed to a review branch. Merge its review request before running the configured branch.' : result.error || 'The commit is not confirmed. Retry to reconcile the same operation.');
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Git write failed. Pending edits remain saved.');
    }
    finally {
        this.busy.set(false);
    } }
    async loadHistory(more = false): Promise<void> {
        this.busy.set(true); this.error.set('');
        try { const page = await this.client.call('listRepositoryMetadataHistory', { ...this.scope(), query: { limit: 20, cursor: more ? this.history().cursor : undefined } }); this.history.set({ ...page, items: more ? [...this.history().items, ...page.items] : page.items }); }
        catch (error) { this.error.set(error instanceof Error ? error.message : 'Metadata history could not be loaded.'); }
        finally { this.busy.set(false); }
    }
    restore(snapshot: SourceRepositoryMetadataRecord): void {
        if (this.text.dirty && !window.confirm('Replace unsaved edits with this retained metadata snapshot?')) return;
        this.text.setValue(JSON.stringify({ manifest: snapshot.data.manifest, catalog: snapshot.data.catalog }, null, 2)); this.text.markAsDirty(); this.review.set(false); this.editingTest.set(null); this.message.set('Retained metadata loaded for review. Save creates a new revision. Earlier snapshots stay unchanged.');
    }
    async loadRemote(): Promise<void> { this.busy.set(true); this.error.set(''); try {
        this.remote.set(await this.client.call('readRemoteRepositoryMetadata', this.scope()));
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Remote metadata could not be read.');
    }
    finally {
        this.busy.set(false);
    } }
    useRemoteBase(): void { const remote = this.remote(); if (remote?.base && window.confirm('Use this Git base after reviewing its metadata against your local changes? Your pending JSON remains unchanged.')) {
        this.reviewedBase.set(remote.base);
        this.text.markAsDirty();
        this.review.set(false);
    } }
    async sync(): Promise<void> { this.busy.set(true); this.error.set(''); try {
        await this.client.call('enqueueRepositorySync', { ...this.scope(), key: randomId(), body: { force: false } });
        this.text.markAsPristine();
        await this.router.navigate(['/projects', this.project, 'sources'], { queryParams: { repository: this.source } });
    }
    catch (error) {
        this.error.set(error instanceof Error ? error.message : 'Sync could not be queued.');
    }
    finally {
        this.busy.set(false);
    } }
}
