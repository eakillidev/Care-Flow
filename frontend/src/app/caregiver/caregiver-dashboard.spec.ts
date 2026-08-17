import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, Subject, throwError } from 'rxjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthService } from '../auth/auth.service';
import { Visit } from '../coordinator/models';
import { CaregiverApiService } from './caregiver-api.service';
import { CaregiverDashboard } from './caregiver-dashboard';
import { GeolocationService, LocationError } from './geolocation.service';

const scheduled: Visit = { id: 'v1', patient: { id: 'p1', first_name: 'Jane', last_name: 'Smith', address: '100 Main Street' }, caregiver: null, scheduled_start: '2026-08-15T13:00:00Z', scheduled_end: '2026-08-15T14:00:00Z', actual_check_in: null, actual_check_out: null, status: 'scheduled', evv_status: 'pending', evv_exception: null, evv: { status: 'pending', exception_reasons: [], check_in: { timestamp: null, latitude: null, longitude: null, distance_from_patient_meters: null }, check_out: { timestamp: null, latitude: null, longitude: null, distance_from_patient_meters: null } } };
const completed: Visit = { ...scheduled, status: 'completed', evv_status: 'exception', actual_check_in: '2026-08-15T13:01:00Z', actual_check_out: '2026-08-15T14:00:00Z', evv_exception: 'outside_geofence', evv: { ...scheduled.evv, status: 'exception', exception_reasons: ['outside_geofence'] } };

describe('CaregiverDashboard', () => {
  let fixture: ComponentFixture<CaregiverDashboard>;
  const api: any = {};
  const location = { current: vi.fn() };
  let visits$: Subject<Visit[]>;
  beforeEach(async () => {
    visits$ = new Subject<Visit[]>();
    api.listVisits = vi.fn(() => visits$); api.getVisit = vi.fn(() => of(scheduled)); api.checkIn = vi.fn(() => of({})); api.checkOut = vi.fn(() => of({}));
    location.current.mockReset(); location.current.mockResolvedValue({ latitude: 39.29, longitude: -76.61 });
    await TestBed.configureTestingModule({ imports: [CaregiverDashboard], providers: [{ provide: CaregiverApiService, useValue: api }, { provide: GeolocationService, useValue: location }, { provide: AuthService, useValue: { logout: vi.fn() } }] }).compileComponents();
    fixture = TestBed.createComponent(CaregiverDashboard);
  });
  async function render(): Promise<void> { fixture.changeDetectorRef.markForCheck(); await fixture.whenStable(); }

  it('renders assigned visits, statuses, and scheduled check-in action', async () => {
    fixture.detectChanges(); visits$.next([scheduled]); await render(); fixture.componentInstance.selected = scheduled; await render();
    expect(fixture.nativeElement.textContent).toContain('Jane Smith');
    expect(fixture.nativeElement.textContent).toContain('EVV pending');
    expect(fixture.nativeElement.textContent).toContain('Check In');
  });
  it('renders the empty and API error states', async () => {
    fixture.detectChanges(); visits$.next([]); await render(); expect(fixture.nativeElement.textContent).toContain('No assigned visits');
    api.listVisits.mockReturnValueOnce(throwError(() => new Error('fail'))); fixture.componentInstance.load(); await render(); expect(fixture.nativeElement.textContent).toContain('Unable to load your visits');
  });
  it('renders completed detail with a human-readable exception', async () => {
    fixture.detectChanges(); visits$.next([completed]); await render(); fixture.componentInstance.selected = completed; await render();
    expect(fixture.nativeElement.textContent).toContain('Outside allowed location');
    expect(fixture.nativeElement.textContent).toContain('completed');
  });
  it('sends browser coordinates and refreshes after check-in', async () => {
    fixture.detectChanges(); await render(); api.listVisits.mockReturnValue(of([scheduled])); fixture.componentInstance.selected = scheduled;
    await fixture.componentInstance.checkIn();
    expect(api.checkIn).toHaveBeenCalledWith('v1', { latitude: 39.29, longitude: -76.61 });
    expect(api.getVisit).toHaveBeenCalledWith('v1');
  });
  it('prevents duplicate check-in while a request is pending', async () => {
    fixture.detectChanges(); await render(); const pending = new Subject(); api.checkIn.mockReturnValue(pending); api.listVisits.mockReturnValue(of([scheduled])); fixture.componentInstance.selected = scheduled;
    const first = fixture.componentInstance.checkIn(); await Promise.resolve(); await fixture.componentInstance.checkIn();
    expect(api.checkIn).toHaveBeenCalledTimes(1); pending.next({}); pending.complete(); await first;
  });
  it('shows permission-denied location errors without calling the API', async () => {
    fixture.detectChanges(); await render(); location.current.mockRejectedValue(new LocationError(1)); fixture.componentInstance.selected = scheduled;
    await fixture.componentInstance.checkIn(); await render();
    expect(api.checkIn).not.toHaveBeenCalled(); expect(fixture.nativeElement.textContent).toContain('Location permission is required');
  });
  it('shows checkout for in-progress visits and handles location failure', async () => {
    fixture.detectChanges(); await render(); const visit = { ...scheduled, status: 'in_progress' as const }; fixture.componentInstance.selected = visit; await render(); expect(fixture.nativeElement.textContent).toContain('Check Out');
    location.current.mockRejectedValue(new LocationError(2)); await fixture.componentInstance.checkOut(); await render(); expect(api.checkOut).not.toHaveBeenCalled(); expect(fixture.nativeElement.textContent).toContain('Unable to determine');
  });
  it('sends current coordinates and renders completed state after checkout', async () => {
    fixture.detectChanges(); await render();
    const inProgress = { ...scheduled, status: 'in_progress' as const };
    api.getVisit.mockReturnValue(of(completed)); api.listVisits.mockReturnValue(of([completed])); fixture.componentInstance.selected = inProgress;
    await fixture.componentInstance.checkOut(); await render();
    expect(api.checkOut).toHaveBeenCalledWith('v1', { latitude: 39.29, longitude: -76.61 });
    expect(fixture.componentInstance.selected?.status).toBe('completed');
    expect(fixture.nativeElement.textContent).toContain('completed');
  });
});
