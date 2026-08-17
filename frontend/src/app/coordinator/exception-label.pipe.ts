import { Pipe, PipeTransform } from '@angular/core';
const labels: Record<string, string> = { early_check_in: 'Early check-in', late_check_in: 'Late check-in', outside_geofence: 'Outside allowed location', checkout_outside_geofence: 'Checkout outside allowed location' };
@Pipe({ name: 'exceptionLabel' })
export class ExceptionLabelPipe implements PipeTransform { transform(reason: string): string { return labels[reason] ?? reason.replaceAll('_', ' '); } }
