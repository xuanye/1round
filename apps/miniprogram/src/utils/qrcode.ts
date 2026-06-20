const QR_CAPACITIES_L = [17, 32, 53, 78, 106, 134];
const PATTERN_POSITION_TABLE = [
  [],
  [6, 18],
  [6, 22],
  [6, 26],
  [6, 30],
  [6, 34],
];
const G15 = 0x0537;
const G15_MASK = 0x5412;

const EXP_TABLE = new Array<number>(256);
const LOG_TABLE = new Array<number>(256);

let gfX = 1;
for (let i = 0; i < 256; i++) {
  EXP_TABLE[i] = gfX;
  LOG_TABLE[gfX] = i;
  gfX <<= 1;
  if (gfX & 0x100) {
    gfX ^= 0x11d;
  }
}

function glog(n: number): number {
  if (n < 1) {
    throw new Error(`glog(${n})`);
  }
  return LOG_TABLE[n];
}

function gexp(n: number): number {
  while (n < 0) n += 255;
  while (n >= 256) n -= 255;
  return EXP_TABLE[n];
}

function getBCHDigit(data: number): number {
  let digit = 0;
  while (data !== 0) {
    digit++;
    data >>>= 1;
  }
  return digit;
}

function getBCHTypeInfo(data: number): number {
  let d = data << 10;
  while (getBCHDigit(d) - getBCHDigit(G15) >= 0) {
    d ^= G15 << (getBCHDigit(d) - getBCHDigit(G15));
  }
  return ((data << 10) | d) ^ G15_MASK;
}

function getMask(maskPattern: number, i: number, j: number): boolean {
  switch (maskPattern) {
    case 0:
      return (i + j) % 2 === 0;
    default:
      throw new Error(`unsupported mask pattern: ${maskPattern}`);
  }
}

function toUTF8Bytes(text: string): number[] {
  const bytes: number[] = [];
  for (let i = 0; i < text.length; i++) {
    let codePoint = text.charCodeAt(i);

    if (codePoint >= 0xd800 && codePoint <= 0xdbff && i + 1 < text.length) {
      const next = text.charCodeAt(i + 1);
      if (next >= 0xdc00 && next <= 0xdfff) {
        codePoint = 0x10000 + ((codePoint - 0xd800) << 10) + (next - 0xdc00);
        i++;
      }
    }

    if (codePoint <= 0x7f) {
      bytes.push(codePoint);
    } else if (codePoint <= 0x7ff) {
      bytes.push(0xc0 | (codePoint >>> 6));
      bytes.push(0x80 | (codePoint & 0x3f));
    } else if (codePoint <= 0xffff) {
      bytes.push(0xe0 | (codePoint >>> 12));
      bytes.push(0x80 | ((codePoint >>> 6) & 0x3f));
      bytes.push(0x80 | (codePoint & 0x3f));
    } else {
      bytes.push(0xf0 | (codePoint >>> 18));
      bytes.push(0x80 | ((codePoint >>> 12) & 0x3f));
      bytes.push(0x80 | ((codePoint >>> 6) & 0x3f));
      bytes.push(0x80 | (codePoint & 0x3f));
    }
  }
  return bytes;
}

function pickTypeNumber(byteLength: number): number {
  const version = QR_CAPACITIES_L.findIndex((capacity) => byteLength <= capacity);
  if (version === -1) {
    throw new Error('invite QR content too long');
  }
  return version + 1;
}

class QRPolynomial {
  private num: number[];

  constructor(num: number[], shift = 0) {
    let offset = 0;
    while (offset < num.length && num[offset] === 0) {
      offset++;
    }

    this.num = new Array(num.length - offset + shift);
    for (let i = 0; i < num.length - offset; i++) {
      this.num[i] = num[i + offset];
    }
  }

  get(index: number): number {
    return this.num[index];
  }

  getLength(): number {
    return this.num.length;
  }

  multiply(other: QRPolynomial): QRPolynomial {
    const num = new Array(this.getLength() + other.getLength() - 1).fill(0);

    for (let i = 0; i < this.getLength(); i++) {
      for (let j = 0; j < other.getLength(); j++) {
        num[i + j] ^= gexp(glog(this.get(i)) + glog(other.get(j)));
      }
    }

    return new QRPolynomial(num);
  }

  mod(other: QRPolynomial): QRPolynomial {
    if (this.getLength() - other.getLength() < 0) {
      return this;
    }

    const ratio = glog(this.get(0)) - glog(other.get(0));
    const num = this.num.slice();

    for (let i = 0; i < other.getLength(); i++) {
      num[i] ^= gexp(glog(other.get(i)) + ratio);
    }

    return new QRPolynomial(num).mod(other);
  }
}

function getErrorCorrectPolynomial(errorCorrectLength: number): QRPolynomial {
  let poly = new QRPolynomial([1]);
  for (let i = 0; i < errorCorrectLength; i++) {
    poly = poly.multiply(new QRPolynomial([1, gexp(i)]));
  }
  return poly;
}

class QRBitBuffer {
  private buffer: number[] = [];
  private length = 0;

  get(index: number): boolean {
    const bufIndex = Math.floor(index / 8);
    return ((this.buffer[bufIndex] >>> (7 - (index % 8))) & 1) === 1;
  }

  put(num: number, length: number): void {
    for (let i = 0; i < length; i++) {
      this.putBit(((num >>> (length - i - 1)) & 1) === 1);
    }
  }

  putBit(bit: boolean): void {
    const bufIndex = Math.floor(this.length / 8);
    if (this.buffer.length <= bufIndex) {
      this.buffer.push(0);
    }

    if (bit) {
      this.buffer[bufIndex] |= 0x80 >>> (this.length % 8);
    }

    this.length++;
  }

  getLengthInBits(): number {
    return this.length;
  }

  toByteArray(byteCount: number): number[] {
    const bytes = new Array(byteCount).fill(0);
    for (let i = 0; i < byteCount; i++) {
      let value = 0;
      for (let j = 0; j < 8; j++) {
        const index = i * 8 + j;
        if (index < this.length && this.get(index)) {
          value |= 0x80 >>> j;
        }
      }
      bytes[i] = value;
    }
    return bytes;
  }
}

function getRSBlocks(typeNumber: number): Array<{ totalCount: number; dataCount: number }> {
  switch (typeNumber) {
    case 1:
      return [{ totalCount: 26, dataCount: 19 }];
    case 2:
      return [{ totalCount: 44, dataCount: 34 }];
    case 3:
      return [{ totalCount: 70, dataCount: 55 }];
    case 4:
      return [{ totalCount: 100, dataCount: 80 }];
    case 5:
      return [{ totalCount: 134, dataCount: 108 }];
    case 6:
      return [
        { totalCount: 86, dataCount: 68 },
        { totalCount: 86, dataCount: 68 },
      ];
    default:
      throw new Error(`unsupported qr version: ${typeNumber}`);
  }
}

function createData(typeNumber: number, dataBytes: number[]): number[] {
  const rsBlocks = getRSBlocks(typeNumber);
  const buffer = new QRBitBuffer();

  buffer.put(4, 4);
  buffer.put(dataBytes.length, 8);
  for (const b of dataBytes) {
    buffer.put(b, 8);
  }

  const totalDataCount = rsBlocks.reduce((sum, block) => sum + block.dataCount, 0);
  const totalBits = totalDataCount * 8;

  if (buffer.getLengthInBits() > totalBits) {
    throw new Error('invite QR content too long');
  }

  if (buffer.getLengthInBits() + 4 <= totalBits) {
    buffer.put(0, 4);
  }

  while (buffer.getLengthInBits() % 8 !== 0) {
    buffer.putBit(false);
  }

  while (buffer.getLengthInBits() < totalBits) {
    buffer.put(0xec, 8);
    if (buffer.getLengthInBits() < totalBits) {
      buffer.put(0x11, 8);
    }
  }

  const data = buffer.toByteArray(totalDataCount);
  const dcdata: number[][] = [];
  const ecdata: number[][] = [];
  let offset = 0;
  let maxDcCount = 0;
  let maxEcCount = 0;

  for (const block of rsBlocks) {
    const dcCount = block.dataCount;
    const ecCount = block.totalCount - dcCount;
    maxDcCount = Math.max(maxDcCount, dcCount);
    maxEcCount = Math.max(maxEcCount, ecCount);

    const currentDc = data.slice(offset, offset + dcCount);
    offset += dcCount;
    dcdata.push(currentDc);

    const rsPoly = getErrorCorrectPolynomial(ecCount);
    const rawPoly = new QRPolynomial(currentDc, rsPoly.getLength() - 1);
    const modPoly = rawPoly.mod(rsPoly);
    const currentEc = new Array(ecCount).fill(0);

    for (let i = 0; i < ecCount; i++) {
      const modIndex = i + modPoly.getLength() - ecCount;
      currentEc[i] = modIndex >= 0 ? modPoly.get(modIndex) : 0;
    }

    ecdata.push(currentEc);
  }

  const result: number[] = [];
  for (let i = 0; i < maxDcCount; i++) {
    for (let r = 0; r < dcdata.length; r++) {
      if (i < dcdata[r].length) {
        result.push(dcdata[r][i]);
      }
    }
  }

  for (let i = 0; i < maxEcCount; i++) {
    for (let r = 0; r < ecdata.length; r++) {
      if (i < ecdata[r].length) {
        result.push(ecdata[r][i]);
      }
    }
  }

  return result;
}

class QRCodeModel {
  private readonly typeNumber: number;
  private readonly dataBytes: number[];
  private readonly moduleCount: number;
  private modules: Array<Array<boolean | null>>;

  constructor(typeNumber: number, dataBytes: number[]) {
    this.typeNumber = typeNumber;
    this.dataBytes = dataBytes;
    this.moduleCount = typeNumber * 4 + 17;
    this.modules = new Array(this.moduleCount);
    for (let row = 0; row < this.moduleCount; row++) {
      this.modules[row] = new Array(this.moduleCount).fill(null);
    }
  }

  make(): void {
    this.setupPositionProbePattern(0, 0);
    this.setupPositionProbePattern(this.moduleCount - 7, 0);
    this.setupPositionProbePattern(0, this.moduleCount - 7);
    this.setupPositionAdjustPattern();
    this.setupTimingPattern();
    this.setupTypeInfo(0);
    this.mapData(createData(this.typeNumber, this.dataBytes), 0);
  }

  isDark(row: number, col: number): boolean {
    return this.modules[row][col] === true;
  }

  getModuleCount(): number {
    return this.moduleCount;
  }

  private setupPositionProbePattern(row: number, col: number): void {
    for (let r = -1; r <= 7; r++) {
      const currentRow = row + r;
      if (currentRow < 0 || currentRow >= this.moduleCount) continue;

      for (let c = -1; c <= 7; c++) {
        const currentCol = col + c;
        if (currentCol < 0 || currentCol >= this.moduleCount) continue;

        if (
          (0 <= r && r <= 6 && (c === 0 || c === 6)) ||
          (0 <= c && c <= 6 && (r === 0 || r === 6)) ||
          (2 <= r && r <= 4 && 2 <= c && c <= 4)
        ) {
          this.modules[currentRow][currentCol] = true;
        } else {
          this.modules[currentRow][currentCol] = false;
        }
      }
    }
  }

  private setupPositionAdjustPattern(): void {
    const positions = PATTERN_POSITION_TABLE[this.typeNumber - 1] || [];

    for (const row of positions) {
      for (const col of positions) {
        if (this.modules[row][col] !== null) {
          continue;
        }

        for (let r = -2; r <= 2; r++) {
          for (let c = -2; c <= 2; c++) {
            this.modules[row + r][col + c] =
              Math.abs(r) === 2 || Math.abs(c) === 2 || (r === 0 && c === 0);
          }
        }
      }
    }
  }

  private setupTimingPattern(): void {
    for (let i = 8; i < this.moduleCount - 8; i++) {
      if (this.modules[i][6] === null) {
        this.modules[i][6] = i % 2 === 0;
      }
      if (this.modules[6][i] === null) {
        this.modules[6][i] = i % 2 === 0;
      }
    }
  }

  private setupTypeInfo(maskPattern: number): void {
    const data = (1 << 3) | maskPattern;
    const bits = getBCHTypeInfo(data);

    for (let i = 0; i < 15; i++) {
      const mod = ((bits >> i) & 1) === 1;

      if (i < 6) {
        this.modules[i][8] = mod;
      } else if (i < 8) {
        this.modules[i + 1][8] = mod;
      } else {
        this.modules[this.moduleCount - 15 + i][8] = mod;
      }

      if (i < 8) {
        this.modules[8][this.moduleCount - i - 1] = mod;
      } else if (i < 9) {
        this.modules[8][15 - i] = mod;
      } else {
        this.modules[8][15 - i - 1] = mod;
      }
    }

    this.modules[this.moduleCount - 8][8] = true;
  }

  private mapData(data: number[], maskPattern: number): void {
    let inc = -1;
    let row = this.moduleCount - 1;
    let bitIndex = 7;
    let byteIndex = 0;

    for (let col = this.moduleCount - 1; col > 0; col -= 2) {
      if (col === 6) {
        col--;
      }

      while (true) {
        for (let c = 0; c < 2; c++) {
          const currentCol = col - c;
          if (this.modules[row][currentCol] !== null) {
            continue;
          }

          let dark = false;
          if (byteIndex < data.length) {
            dark = ((data[byteIndex] >>> bitIndex) & 1) === 1;
          }

          if (getMask(maskPattern, row, currentCol)) {
            dark = !dark;
          }

          this.modules[row][currentCol] = dark;
          bitIndex--;

          if (bitIndex === -1) {
            byteIndex++;
            bitIndex = 7;
          }
        }

        row += inc;
        if (row < 0 || row >= this.moduleCount) {
          row -= inc;
          inc = -inc;
          break;
        }
      }
    }
  }
}

export function drawQRCode(canvasId: string, text: string, size: number, pageInstance: any): void {
  const dataBytes = toUTF8Bytes(text);
  const typeNumber = pickTypeNumber(dataBytes.length);
  const qr = new QRCodeModel(typeNumber, dataBytes);
  qr.make();

  const quietZone = Math.max(8, Math.floor(size * 0.08));
  const drawSize = size - quietZone * 2;
  const count = qr.getModuleCount();
  const tile = drawSize / count;
  const ctx = wx.createCanvasContext(canvasId, pageInstance);

  ctx.setFillStyle('#FFFFFF');
  ctx.fillRect(0, 0, size, size);

  ctx.setFillStyle('#1F695D');
  for (let row = 0; row < count; row++) {
    for (let col = 0; col < count; col++) {
      if (!qr.isDark(row, col)) {
        continue;
      }

      const x = quietZone + col * tile;
      const y = quietZone + row * tile;
      const w = Math.ceil((col + 1) * tile) - Math.floor(col * tile);
      const h = Math.ceil((row + 1) * tile) - Math.floor(row * tile);
      ctx.fillRect(Math.floor(x), Math.floor(y), w, h);
    }
  }

  ctx.draw();
}
