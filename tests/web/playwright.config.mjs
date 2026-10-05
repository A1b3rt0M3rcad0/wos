import {defineConfig} from 'playwright/test';
export default defineConfig({
 timeout:120000,expect:{timeout:8000},testDir:'.',testMatch:'*.spec.mjs',workers:1,reporter:'list',
 use:{actionTimeout:10000,headless:true,viewport:{width:1280,height:900},
  ...(process.env.WOS_TEST_CHROMIUM_PATH?{launchOptions:{executablePath:process.env.WOS_TEST_CHROMIUM_PATH,args:['--no-sandbox','--no-zygote','--single-process','--disable-dev-shm-usage']}}:{})
 }
});
