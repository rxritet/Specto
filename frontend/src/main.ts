import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';

async function bootstrap(): Promise<void> {
  try {
    await bootstrapApplication(AppComponent, appConfig);
  } catch (err) {
    console.error(err);
  }
}

// eslint-disable-next-line sonarjs/prefer-top-level-await
void bootstrap();
