import * as path from 'path';
import { expect } from 'chai';
import { setUpPuppeteerAndDemoPageServer, addEventListenersToPuppeteerPage, takeScreenshot } from '../../../puppeteer-tests/util';

describe('changelists-page-sk', () => {
  // Contains page and baseUrl.
  const testBed = setUpPuppeteerAndDemoPageServer(path.join(__dirname, '..', '..', 'webpack.config.ts'));

  beforeEach(async () => {
    const eventPromise = await addEventListenersToPuppeteerPage(testBed.page, ['end-task']);
    const loaded = eventPromise('end-task'); // Emitted when page is loaded.
    await testBed.page.goto(`${testBed.baseUrl}/dist/changelists-page-sk.html`);
    await loaded;
  });

  it('should render the demo page', async () => {
    // Smoke test.
    expect(await testBed.page.$$('changelists-page-sk')).to.have.length(1);
  });

  it('defaults to only open changelists', async () => {
    await testBed.page.setViewport({ width: 1200, height: 500 });
    await takeScreenshot(testBed.page, 'gold', 'changelists-page-sk');
  });

  it('can show all changelists with a click', async () => {
    await testBed.page.setViewport({ width: 1200, height: 600 });
    await testBed.page.click('.controls checkbox-sk');
    await takeScreenshot(testBed.page, 'gold', 'changelists-page-sk_show-all');
  });
});
