import { Injectable } from '@angular/core';

declare let ga: any;

@Injectable()
export class AnalyticsService {
  private enabled: boolean = false;

  constructor() {
    this.enabled = (typeof ga === "function");
    if (!this.enabled) {
      console.warn("ganalytics not loaded.");
    }
  }

  public trackEvent(eventCategory: string, event: string, button: string): void {
    if (this.enabled) {
      try {
        ga('send', 'event', eventCategory, event, button);
      } catch (e) {
        console.error("ga error", e);
      }
    }
  }

  public trackPage(page: string): void {
    if (this.enabled) {
      try {
        ga('send', 'pageview', page);
      } catch (e) {
        console.error("ga error", e);
      }
    }
  }
}
