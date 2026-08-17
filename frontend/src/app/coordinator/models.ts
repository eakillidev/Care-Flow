export type VisitStatus = 'scheduled' | 'in_progress' | 'completed' | 'cancelled';
export type EvvStatus = 'pending' | 'verified' | 'exception';
export interface PersonSummary { id: string; first_name: string; last_name: string; address?: string; email?: string; }
export interface EvvPoint { timestamp: string | null; latitude: number | null; longitude: number | null; distance_from_patient_meters: number | null; }
export interface EvvDetail { status: EvvStatus; exception_reasons: string[]; check_in: EvvPoint; check_out: EvvPoint; }
export interface Visit { id: string; patient: PersonSummary; caregiver: PersonSummary | null; scheduled_start: string; scheduled_end: string; actual_check_in?: string | null; actual_check_out?: string | null; status: VisitStatus; evv_status: EvvStatus; evv_exception: string | null; evv: EvvDetail; }
export interface Summary { total_visits: number; scheduled: number; in_progress: number; completed: number; cancelled: number; evv_verified: number; evv_exceptions: number; }
export interface Filters { status?: string; evv_status?: string; caregiver_id?: string; patient_id?: string; from?: string; to?: string; }
export interface Patient { id: string; first_name: string; last_name: string; }
