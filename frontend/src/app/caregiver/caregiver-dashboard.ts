import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { ExceptionLabelPipe } from '../coordinator/exception-label.pipe';
import { Visit } from '../coordinator/models';
import { AppHeader } from '../shared/app-header';
import { CaregiverApiService } from './caregiver-api.service';
import { GeolocationService, LocationError } from './geolocation.service';

@Component({ selector: 'app-caregiver-dashboard', imports: [CommonModule, ExceptionLabelPipe, AppHeader], templateUrl: './caregiver-dashboard.html', styleUrl: './caregiver-dashboard.css' })
export class CaregiverDashboard implements OnInit {
  visits: Visit[] = [];
  selected?: Visit;
  loading = true;
  error = '';
  actionError = '';
  processing = false;

  constructor(private readonly api: CaregiverApiService, private readonly location: GeolocationService) {}
  ngOnInit(): void { this.load(); }

  load(): void {
    this.loading = true;
    this.error = '';
    this.api.listVisits().subscribe({
      next: visits => { this.visits = visits; this.loading = false; },
      error: () => { this.loading = false; this.error = 'Unable to load your visits.'; }
    });
  }

  open(visit: Visit): void {
    this.actionError = '';
    this.api.getVisit(visit.id).subscribe({ next: detail => this.selected = detail, error: () => this.error = 'Unable to load visit details.' });
  }
  close(): void { if (!this.processing) this.selected = undefined; }

  async checkIn(): Promise<void> { await this.perform('check-in'); }
  async checkOut(): Promise<void> { await this.perform('check-out'); }

  private async perform(action: 'check-in' | 'check-out'): Promise<void> {
    const visit = this.selected;
    if (!visit || this.processing || (action === 'check-in' ? visit.status !== 'scheduled' : visit.status !== 'in_progress')) return;
    this.processing = true;
    this.actionError = '';
    try {
      const coordinates = await this.location.current();
      await firstValueFrom(action === 'check-in' ? this.api.checkIn(visit.id, coordinates) : this.api.checkOut(visit.id, coordinates));
      const [detail, visits] = await Promise.all([firstValueFrom(this.api.getVisit(visit.id)), firstValueFrom(this.api.listVisits())]);
      this.selected = detail;
      this.visits = visits;
    } catch (error) {
      this.actionError = error instanceof LocationError
        ? (error.code === 1 ? `Location permission is required to ${action}.` : 'Unable to determine your current location.')
        : `Unable to ${action} this visit.`;
    } finally {
      this.processing = false;
    }
  }

  name(visit: Visit): string { return `${visit.patient.first_name} ${visit.patient.last_name}`; }
  reasons(visit: Visit): string[] { return visit.evv?.exception_reasons ?? visit.evv_exception?.split(',').filter(Boolean) ?? []; }
}
