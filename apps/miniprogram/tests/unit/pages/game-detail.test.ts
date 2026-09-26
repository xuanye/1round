import { instance, loadSource } from '../../helpers/page';

describe('game-detail page', () => {
  it('parses settlement scene values into share tokens', () => {
    const { page } = loadSource('pages/game-detail/index.ts');
    for (const [scene, expected] of [
      ['Abcdef0123456789_-Abcdef', 'Abcdef0123456789_-Abcdef'],
      ['036662a691264225bc19ab55e1748311', '036662a6-9126-4225-bc19-ab55e1748311'],
      ['shareToken=Abcdef0123456789_-Abcdef', 'Abcdef0123456789_-Abcdef'],
    ] as Array<[string, string]>) {
      const pageInstance = instance(page!);
      pageInstance.onLoad({ scene });
      expect(pageInstance.data.shareToken).toBe(expected);
    }
  });
});
