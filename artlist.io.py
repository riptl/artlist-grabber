#!/usr/bin/env python3

# Credit to HoLLy#2750 for reversing
# Python version by terorie

import requests
import sys
import csv

if len(sys.argv) < 3:
	print("Usage: ./artlist.io.py <min_page> <max_page>", file=sys.stderr)
	exit(1)

min_page = int(sys.argv[1])
max_page = int(sys.argv[2])

csvwriter = csv.writer(sys.stdout)

for i in range(min_page, max_page):
	payload= {
		'searchTerm': '',
		'categoryIDs': '',
		'songSortID': '1',
		'page': str(i),
		'durationMin': 0,
		'durationMax': 0,
		'onlyVocal': '',
	}
	resp = requests.get('https://artlist.io/api/Song/List', params=payload)
	json = resp.json()
	for song in json['songs']:
		csvwriter.writerow([song['artistName'], song['songName'], song['MP3FilePath']])

