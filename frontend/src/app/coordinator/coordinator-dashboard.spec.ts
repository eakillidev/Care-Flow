import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';
import { CoordinatorDashboard } from './coordinator-dashboard';
import { Visit } from './models';
import { VisitApiService } from './visit-api.service';
import { AuthService } from '../auth/auth.service';

const visit: Visit = { id: 'v1', patient: { id: 'p1', first_name: 'Jane', last_name: 'Smith' }, caregiver: { id: 'c1', first_name: 'Maria', last_name: 'Lopez' }, scheduled_start: '2026-08-15T13:00:00Z', scheduled_end: '2026-08-15T14:00:00Z', status: 'completed', evv_status: 'exception', evv_exception: 'late_check_in', evv: { status: 'exception', exception_reasons: ['late_check_in'], check_in: { timestamp: '2026-08-15T13:20:00Z', latitude: 39, longitude: -76, distance_from_patient_meters: 42.7 }, check_out: { timestamp: null, latitude: null, longitude: null, distance_from_patient_meters: null } } };

describe('CoordinatorDashboard', () => {
  let fixture: ComponentFixture<CoordinatorDashboard>;
  const summary = { total_visits: 1, scheduled: 0, in_progress: 0, completed: 1, cancelled: 0, evv_verified: 0, evv_exceptions: 1 };
  const api: any = {};
  beforeEach(async () => { api.caregivers = () => of([]); api.patients = () => of([]); api.listVisits = () => of([visit]); api.summary = () => of(summary); api.getVisit = () => of(visit); await TestBed.configureTestingModule({ imports: [CoordinatorDashboard], providers: [{ provide: VisitApiService, useValue: api }, { provide: AuthService, useValue: { logout: () => undefined } }] }).compileComponents(); fixture = TestBed.createComponent(CoordinatorDashboard); });
  it('loads dashboard and renders EVV exception badge', () => { fixture.detectChanges(); expect(fixture.nativeElement.textContent).toContain('EVV Exceptions'); expect(fixture.nativeElement.textContent).toContain('exception'); });
  it('renders visit detail and human-readable exception reason', () => { fixture.componentInstance.selected = visit; fixture.detectChanges(); expect(fixture.nativeElement.textContent).toContain('Late check-in'); expect(fixture.nativeElement.textContent).toContain('42.7 m'); expect(fixture.nativeElement.textContent).toContain('—'); });
  it('renders filtered empty state', () => { api.listVisits = () => of([]); fixture.componentInstance.filters = { status: 'cancelled' }; fixture.detectChanges(); expect(fixture.nativeElement.textContent).toContain('No visits match'); });
  it('shows API errors', () => { api.listVisits = () => throwError(() => new Error('fail')); fixture.detectChanges(); expect(fixture.nativeElement.textContent).toContain('Unable to load visits'); });
});
