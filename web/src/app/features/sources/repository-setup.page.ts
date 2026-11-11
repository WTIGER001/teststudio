import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { SourcesService } from './sources.service';
import { RepositoriesService } from './repositories.service';
import { RepositoryEditorComponent } from './repository-editor.component';
@Component({ standalone: true, imports: [RouterLink, RepositoryEditorComponent], providers: [SourcesService, RepositoriesService], changeDetection: ChangeDetectionStrategy.OnPush, template: `
 <div class="page-heading"><div><span class="eyebrow">{{ sources.workspace.project()?.name || 'Project' }}</span><h1>{{ route.snapshot.paramMap.get('sourceId') ? 'Edit repository' : 'Connect repository' }}</h1><p class="muted">Choose metadata ownership and review access before saving. Saving never starts tests.</p></div><a class="button secondary" [routerLink]="['/projects', sources.project(), 'sources']">Back to sources</a></div>
 @if (service.loading()) { <p role="status">Loading repository options…</p> }
 @if (service.error() && !service.editor()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
 @if (service.editor()) { <ts-repository-editor /> }
 `, styles: [':host { display: block; max-width: 56rem; margin-inline: auto; }'] })
export class RepositorySetupPage {
    readonly sources = inject(SourcesService);
    readonly service = inject(RepositoriesService);
    readonly route = inject(ActivatedRoute);
    constructor() { void this.open(); }
    private async open(): Promise<void> { await this.service.load(); const id = this.route.snapshot.paramMap.get('sourceId'); if (id) {
        const row = await this.service.readEditor(id);
        if (row)
            this.service.edit(row);
    }
    else
        this.service.edit(); }
    canLeave(): boolean { return !this.service.form.dirty || window.confirm('This repository has unsaved changes. Leave without saving?'); }
}
