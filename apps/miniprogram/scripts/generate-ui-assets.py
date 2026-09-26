"""Generate transparent UI assets; requires Pillow only for asset authoring."""
from pathlib import Path
import math
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1] / 'src' / 'images'
GREEN, GRAY, RED, PAPER = '#175D50', '#68716D', '#B95740', '#F8F7F3'
SCALE = 4


def canvas(width=24, height=24):
    image = Image.new('RGBA', (width * SCALE, height * SCALE))
    return image, ImageDraw.Draw(image)


def line(draw, points, color, width=1.8):
    points = [(round(x * SCALE), round(y * SCALE)) for x, y in points]
    draw.line(points, fill=color, width=round(width * SCALE), joint='curve')
    radius = width * SCALE / 2
    for x, y in [points[0], points[-1]]:
        draw.ellipse((x-radius, y-radius, x+radius, y+radius), fill=color)


def rounded(draw, bounds, color, width=1.8, fill=None, radius=2):
    draw.rounded_rectangle(tuple(round(n*SCALE) for n in bounds), radius=round(radius*SCALE), fill=fill, outline=color, width=round(width*SCALE))


def save(image, relative):
    path = ROOT / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    image.save(path)


def heart_points():
    points = []
    for step in range(121):
        t = step * math.tau / 120
        x = 16 * math.sin(t)**3
        y = 13*math.cos(t)-5*math.cos(2*t)-2*math.cos(3*t)-math.cos(4*t)
        points.append((x/34+.5, .48-y/34))
    return points


def suit(draw, kind, x, y, size, color):
    if kind in ('heart', 'spade'):
        points = heart_points()
        if kind == 'spade': points = [(px, 1-py) for px, py in points]
        draw.polygon([((x+px*size)*SCALE,(y+py*size)*SCALE) for px,py in points], fill=color)
    elif kind == 'diamond':
        draw.polygon([((x+px*size)*SCALE,(y+py*size)*SCALE) for px,py in [(0.5,0),(1,.5),(.5,1),(0,.5)]],fill=color)
    else:
        for cx,cy in [(.5,.25),(.25,.55),(.75,.55)]:
            draw.ellipse(((x+(cx-.24)*size)*SCALE,(y+(cy-.24)*size)*SCALE,(x+(cx+.24)*size)*SCALE,(y+(cy+.24)*size)*SCALE),fill=color)
    if kind in ('spade','club'):
        draw.polygon([((x+px*size)*SCALE,(y+py*size)*SCALE) for px,py in [(.5,.6),(.32,1),(.68,1)]],fill=color)


for active, color in [(False, GRAY), (True, GREEN)]:
    suffix = '-active' if active else ''
    image, draw = canvas()
    for bounds, angle, offset in [((0,0,11,16),16,(2,3)),((0,0,11,16),-14,(11,5))]:
        card, cd = canvas(14,19)
        rounded(cd,(1,1,12,17),color,width=1.6,radius=1.8)
        card = card.rotate(angle, resample=Image.Resampling.BICUBIC, expand=True)
        image.alpha_composite(card,(round(offset[0]*SCALE)-8,round(offset[1]*SCALE)-8))
    save(image, f'tabbar/game{suffix}.png')
    image, draw = canvas()
    for x, top in [(5,13),(12,4),(19,9)]: line(draw,[(x,top),(x,21)],color)
    save(image,f'tabbar/records{suffix}.png')
    image, draw = canvas()
    draw.ellipse((8*SCALE,2*SCALE,16*SCALE,10*SCALE),outline=color,width=round(1.8*SCALE))
    rounded(draw,(3,13,21,22),color,width=1.8,radius=4)
    save(image,f'tabbar/mine{suffix}.png')

image, draw = canvas(32,32)
for name,x,y,color in [('spade',0,0,GREEN),('heart',17,0,RED),('diamond',0,17,RED),('club',17,17,GREEN)]:
    suit(draw,name,x,y,14,color)
save(image,'home/suits.png')

for name,color in [('plus',GREEN),('plus-white','#FFFFFF')]:
    image,draw=canvas()
    line(draw,[(12,3),(12,21)],color)
    line(draw,[(3,12),(21,12)],color)
    save(image,f'home/{name}.png')
image,draw=canvas()
for pts in [[(3,8),(3,3),(8,3)],[(16,3),(21,3),(21,8)],[(21,16),(21,21),(16,21)],[(8,21),(3,21),(3,16)]]:
    line(draw,pts,'#FFFFFF')
line(draw,[(6,12),(18,12)],'#FFFFFF',1.6)
save(image,'home/scan.png')
image,draw=canvas()
for x,y in [(2,2),(14,2),(2,14)]: rounded(draw,(x,y,x+7,y+7),GREEN,1.5,radius=1)
for x,y in [(15,15),(20,14),(14,20),(20,20)]:
    draw.rectangle((x*SCALE,y*SCALE,(x+2)*SCALE,(y+2)*SCALE),fill=GREEN)
save(image,'home/qr.png')

image,draw=canvas(124,144)
font_path = Path('/System/Library/Fonts/Supplemental/Arial.ttf')
font = ImageFont.truetype(str(font_path),16*SCALE) if font_path.exists() else ImageFont.load_default(size=16*SCALE)
for x,y,angle,label,shape,color in [(48,34,-15,'K','heart',RED),(8,8,12,'A','spade',GRAY)]:
    card,cd=canvas(53,86)
    rounded(cd,(1,1,51,84),'#899A93',.9,PAPER,5)
    cd.text((7*SCALE,6*SCALE),label,font=font,fill=color)
    suit(cd,shape,15,35,24,color)
    card=card.rotate(angle,resample=Image.Resampling.BICUBIC,expand=True)
    image.alpha_composite(card,(x*SCALE,y*SCALE))
save(image,'home/playing-cards.png')
