import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { ExceptionLabelPipe } from './exception-label.pipe';
import { Filters, Patient, PersonSummary, Summary, Visit } from './models';
import { VisitApiService } from './visit-api.service';
import { AppHeader } from '../shared/app-header';

@Component({ selector: 'app-coordinator-dashboard', imports: [CommonModule, FormsModule, ExceptionLabelPipe, AppHeader], templateUrl: './coordinator-dashboard.html', styleUrl: './coordinator-dashboard.css' })
export class CoordinatorDashboard implements OnInit {
  filters: Filters = {}; visits: Visit[] = []; caregivers: PersonSummary[] = []; patients: Patient[] = []; summary?: Summary; selected?: Visit; loading = true; error = ''; hasFilters = false;
  constructor(private readonly api: VisitApiService) {}
  ngOnInit() { forkJoin({ caregivers: this.api.caregivers(), patients: this.api.patients() }).subscribe({ next: data => { this.caregivers = data.caregivers; this.patients = data.patients; this.load(); }, error: () => { this.loading = false; this.error = 'Unable to load dashboard data.'; } }); }
  load() { this.loading = true; this.error = ''; this.hasFilters = Object.values(this.filters).some(Boolean); forkJoin({ visits: this.api.listVisits(this.filters), summary: this.api.summary({ from: this.filters.from, to: this.filters.to }) }).subscribe({ next: data => { this.visits = data.visits; this.summary = data.summary; this.loading = false; }, error: () => { this.loading = false; this.error = 'Unable to load visits.'; } }); }
  reset() { this.filters = {}; this.load(); }
  open(visit: Visit) { this.api.getVisit(visit.id).subscribe({ next: value => this.selected = value, error: () => this.error = 'Unable to load visit details.' }); }
  close() { this.selected = undefined; }
  name(person: PersonSummary | Patient) { return `${person.first_name} ${person.last_name}`; }
}
