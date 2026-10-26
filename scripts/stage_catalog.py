#!/usr/bin/env python3
"""Stage a pinned PokéAPI official v2 CSV snapshot, without opening any database.
Caches sequential requests; --offline reproduces the same canonical output.
"""
import argparse,csv,datetime,hashlib,io,json,pathlib,re,time,urllib.request
FILES=['pokemon_species.csv','pokemon.csv','pokemon_forms.csv','pokemon_species_names.csv','pokemon_form_names.csv','languages.csv','LICENSE.md']

def build(cache, revision):
    docs={}; tables={}
    for name in FILES:
        raw=(cache/name).read_bytes()
        docs[name]={'url':f'https://raw.githubusercontent.com/PokeAPI/pokeapi/{revision}/'+('data/v2/csv/'+name if name.endswith('.csv') else name),'sha256':hashlib.sha256(raw).hexdigest()}
        if name.endswith('.csv'):tables[name]=list(csv.DictReader(io.StringIO(raw.decode())))
    langs={r['id']:r['identifier'] for r in tables['languages.csv']}
    species={int(r['id']):r for r in tables['pokemon_species.csv']}
    pokemon={int(r['id']):r for r in tables['pokemon.csv']}
    names={}; form_names={}
    def add(target,id,lang,value):
        values=target.setdefault(id,{})
        if lang in values and values[lang]!=value:raise ValueError('conflicting localized name; no snapshot produced')
        values[lang]=value
    for r in tables['pokemon_species_names.csv']:
        add(names,int(r['pokemon_species_id']),langs[r['local_language_id']],r['name'])
    for r in tables['pokemon_form_names.csv']:
        # Only full pokemon names count as localized form names. Never invent a translation.
        if r['pokemon_name']:add(form_names,int(r['pokemon_form_id']),langs[r['local_language_id']],r['pokemon_name'])
    result=[]
    for id,r in sorted(species.items()):
        result.append(dict(kind='species',external_id=id,species_external_id=id,national_dex=id,slug=r['identifier'],names=names.get(id,{}),inclusion='included',reason='national-species',csv_file='pokemon_species.csv'))
    # Conservative policy: known persistent families are explicit; unreviewed variants stay unknown.
    persistent={201,412,413,422,423,479,550,585,586,666,669,670,671,676,678,710,711,741,745,774,849,869,876,892,902,916,925,964,978}
    for r in tables['pokemon_forms.csv']:
        id=int(r['id']); p=pokemon[int(r['pokemon_id'])]; sid=int(p['species_id']); slug=r['identifier']; fi=r['form_identifier']
        if sid not in species:raise ValueError('missing species mapping')
        inclusion,reason='unknown','unreviewed-variant'
        if r['is_battle_only']=='1' or r['is_mega']=='1':inclusion,reason='excluded','temporary-battle-form'
        elif sid==25 and fi:inclusion,reason='excluded','costume-or-special-pikachu'
        elif p['is_default']=='1' and r['is_default']=='1' and not fi:inclusion,reason='included','default-persistent-form'
        elif fi in {'alola','galar','hisui','paldea','paldea-combat-breed','paldea-blaze-breed','paldea-aqua-breed'}:inclusion,reason='included','regional-persistent-form'
        elif sid in persistent:inclusion,reason='included','explicit-persistent-family-v1'
        n=form_names.get(id,{})
        if not fi and p['is_default']=='1':n=names.get(sid,{})
        result.append(dict(kind='form',external_id=id,species_external_id=sid,national_dex=0,slug=slug,names=n,inclusion=inclusion,reason=reason,csv_file='pokemon_forms.csv'))
    fetched=(cache/'fetched_at').read_text().strip()
    return dict(format=1,source='pokeapi-official-v2-csv',revision=revision,fetched_at=fetched,documents=docs,records=result)

def main():
    a=argparse.ArgumentParser();a.add_argument('--cache',required=True,type=pathlib.Path);a.add_argument('--revision',required=True);a.add_argument('--offline',action='store_true');a.add_argument('--output',required=True,type=pathlib.Path);args=a.parse_args()
    if not re.fullmatch('[a-f0-9]{40}',args.revision):a.error('revision must be a full SHA')
    args.cache.mkdir(parents=True,exist_ok=True)
    # Do not reuse a cache belonging to a different revision.
    rev=args.cache/'revision'
    if rev.exists() and rev.read_text().strip()!=args.revision:a.error('cache revision mismatch')
    if not rev.exists():
        if args.offline:a.error('offline cache has no revision')
        rev.write_text(args.revision)
    for name in FILES:
        path=args.cache/name
        if path.exists():continue
        if args.offline:a.error('missing offline cache: '+name)
        url=f'https://raw.githubusercontent.com/PokeAPI/pokeapi/{args.revision}/'+('data/v2/csv/'+name if name.endswith('.csv') else name)
        req=urllib.request.Request(url,headers={'User-Agent':'PokemonShinyTracker/0.4 (+https://github.com/zl14229/pokemon-shiny-tracker)'})
        raw=urllib.request.urlopen(req,timeout=30).read(4<<20)
        with path.open('xb') as f:f.write(raw)
        time.sleep(1)
    stamp=args.cache/'fetched_at'
    if not stamp.exists():
        if args.offline:a.error('offline cache has no timestamp')
        stamp.write_text(datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00','Z'))
    snapshot=build(args.cache,args.revision)
    with args.output.open('x') as f:json.dump(snapshot,f,ensure_ascii=False,separators=(',',':'),sort_keys=True);f.write('\n')
    print(json.dumps({'species':sum(r['kind']=='species' for r in snapshot['records']),'forms':sum(r['kind']=='form' for r in snapshot['records']),'revision':args.revision}))
if __name__=='__main__':main()
