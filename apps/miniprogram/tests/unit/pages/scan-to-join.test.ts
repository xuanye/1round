import { loadSource } from '../../helpers/page';

describe('home scanToJoin', () => {
  it('normalizes scanned scene values into an invite-code route', () => {
    let scanResult: Record<string, string> | undefined;
    let navigatedTo: string | undefined;
    let toast: string | undefined;
    const wx = {
      scanCode({ success }: { success: (value: Record<string, string>) => void }) {
        success(scanResult as Record<string, string>);
      },
      navigateTo({ url }: { url: string }) { navigatedTo = url; },
      showToast({ title }: { title: string }) { toast = title; },
    };
    const { page } = loadSource('pages/home/index.ts', { wx });
    expect(page).toBeDefined();

    for (const [label, result] of [
      ['unescaped scene', { path: 'pages/game-join/index?scene=code=ABC123' }],
      ['encoded scene', { path: 'pages/game-join/index?scene=code%3DABC123' }],
      ['direct invite link', { result: 'pages/game-join/index?inviteCode=ABC123' }],
    ] as Array<[string, Record<string, string>]>) {
      scanResult = result;
      navigatedTo = undefined;
      toast = undefined;
      page!.scanToJoin();
      expect(navigatedTo).toBe('/pages/game-join/index?inviteCode=ABC123');
      expect(toast).toBeUndefined();
    }
  });
});
