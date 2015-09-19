/* vi:set ts=8 sts=8 sw=8 noet:
 *
 * PMS  <<Practical Music Search>>
 * Copyright (C) 2006-2015  Kim Tore Jensen
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 * display.cpp - ncurses, display and window management
 *
 */

#include "display.h"
#include "config.h"
#include "pms.h"
#include <cstdio>
#include <sstream>

extern Pms *			pms;

Point::Point()
{
	x = 0;
	y = 0;
}

Point::Point(uint16_t x_, uint16_t y_)
{
	x = x_;
	y = y_;
}

Point &
Point::operator=(const Point & src)
{
	x = src.x;
	y = src.y;

	return *this;
}

BBox::BBox()
{
	tl.x = 0;
	tl.y = 0;
	br.x = 0;
	br.y = 0;
	window = NULL;
}

inline
uint16_t
BBox::top()
{
	return tl.y;
}

inline
uint16_t
BBox::bottom()
{
	return br.y;
}

inline
uint16_t
BBox::left()
{
	return tl.x;
}

inline
uint16_t
BBox::right()
{
	return br.x;
}

inline
uint16_t
BBox::width()
{
	return br.x - tl.x;
}

inline
uint16_t
BBox::height()
{
	return br.y - tl.y;
}

bool
BBox::clear(color * c)
{
	int y;
	int w;

	if (c) {
		w = width();
		y = height();
		if (wattron(window, c->pair()) == ERR) {
			return false;
		}
		while (y != 0) {
			if (mvwhline(window, y--, 0, ' ', w) == ERR) {
				return false;
			}
		}
		if (wattroff(window, c->pair()) == ERR) {
			return false;
		}
	} else {
		return (wclear(window) != ERR);
	}
}

bool
BBox::resize(const Point & tl_, const Point & br_)
{
	int rc;

	if (window != NULL) {
		rc = delwin(window);
		assert(rc != ERR);
		if (rc != ERR) {
			return false;
		}
		window = NULL;
	}

	tl = tl_;
	br = br_;

	window = newwin(height(), width(), tl.y, tl.x);
	assert(window != NULL);

	return (window != NULL);
}

bool
BBox::refresh()
{
	return (wrefresh(window) != ERR);
}

/*
 *
 * Display class
 *
 */

Display::Display(Control * n_comm)
{
	//comm = n_comm;
	//curwin = NULL;
	//lastwin = NULL;
}

Display::~Display()
{
	this->uninit();
}

/*
 * Switch mouse support on or off by setting the mouse mask
 */
mmask_t		Display::setmousemask()
{
	if (pms->options->mouse)
		mmask = mousemask(ALL_MOUSE_EVENTS, &oldmmask);
	else
		mmask = mousemask(0, &oldmmask);

	return mmask;
}

/*
 * Initialize ncurses
 */
bool
Display::init()
{
	/* Fetch most keys and turn off echoing */
	initscr();
	raw();
	noecho();
	keypad(stdscr, true);
	setmousemask();

	if (has_colors()) {
		start_color();
		use_default_colors();
	}

	/* Hide cursor */
	curs_set(0);

	resized();

	return true;
}

/*
 * Delete all windows and end ncurses mode
 */
void
Display::uninit()
{
	vector<List *>::iterator i;

	i = lists.begin();

	while (i != lists.end()) {
		delete *i;
		++i;
	}

	lists.clear();

	endwin();
}

/*
 * Resizes windows.
 */
void
Display::resized()
{
	vector<List *>::iterator iter;

	topbar.resize(Point(0, 0), Point(COLS - 1, pms->options->topbar_lines.size() - 1));
	title.resize(Point(0, topbar.bottom() + 1), Point(COLS - 1, topbar.bottom() + 1));
	main_window.resize(Point(0, title.bottom() + 1), Point(COLS - 1, LINES - 2));
	statusbar.resize(Point(0, main_window.bottom() + 1), Point(COLS - 4, main_window.bottom() + 1));
	position_readout.resize(Point(0, statusbar.bottom()), Point(statusbar.right() + 1, statusbar.bottom()));

	/* FIXME
	iter = lists.begin();
	while (iter != lists.end()) {
		(*iter++)->set_column_size();
	}
	*/
}

/*
 * Flushes drawn output to screen for all windows on current screen.
 */
void
Display::refresh()
{
	topbar.refresh();
	title.refresh();
	main_window.refresh();
	statusbar.refresh();
	position_readout.refresh();
}

/*
 * Redraws all visible windows
 */
void		Display::draw()
{
	assert(false);
	/* FIXME */
	/*
	if (curwin && curwin->wantdraw)
		curwin->draw();
	if (topbar->wantdraw)
		topbar->draw();
	if (statusbar->wantdraw)
		statusbar->draw();
	if (positionreadout->wantdraw)
		positionreadout->draw();
	*/
}

/*
 * Redraws all visible windows regardless of state
 */
void		Display::forcedraw()
{
	assert(false);
	/*
	topbar->draw();
	statusbar->draw();
	positionreadout->draw();
	if (curwin) curwin->draw();
	*/
}

/*
 * Set XTerm window title.
 *
 * The current xterm title exists under the WM_NAME property,
 * and can be retrieved with `xprop -notype -id $WINDOWID WM_NAME`.
 */
void
Display::set_xterm_title()
{
	unsigned int	reallen;
	string		title;
	ostringstream	oss;

	if (!pms->options->xtermtitle.size()) {
		return;
	}

	if (getenv("WINDOWID")) {
		title = pms->formatter->format(pms->cursong(), pms->options->xtermtitle, reallen, NULL, true);
		pms->log(MSG_DEBUG, 0, _("Set XTerm window title: '%s'\n"), title.c_str());

		oss << "\033]0;" << title << '\007';
		putp(oss.str().c_str());

		//stdout is in line buffered mode be default and thus needs explicit flush to communicate with terminal successfully.
		fflush(stdout);
	} else {
		pms->log(MSG_DEBUG, 0, _("Disabling XTerm window title: WINDOWID not found.\n"));
		pms->options->xtermtitle = "";
	}
}

Song *
Display::cursorsong()
{
	Songlist * list;

	if ((list = dynamic_cast<Songlist *>(active_list)) == NULL) {
		return NULL;
	}

	return list->cursorsong();
}




/*
 *
 * End of display class.
 *
 * Playlist column class
 *
 */
pms_column::pms_column(string n_title, Item n_type, unsigned int n_minlen)
{
	title	= n_title;
	type	= n_type;
	minlen	= n_minlen;
	abslen	= -1;
	median	= 0;
	items	= 0;
}

void		pms_column::addmedian(unsigned int n)
{
	++items;
	median += n;
	abslen = -1;
}

unsigned int	pms_column::len()
{
	if (abslen < 0)
	{
		if (items == 0)
			abslen = 0;
		else
			abslen = (median / items);
	}
	if ((unsigned int)abslen < minlen) {
		return minlen;
	}

	return (unsigned int)abslen;
}


/*
 * Prints formatted output onto a window. Borders are handled correctly.
 *
 * %s		= char *
 * %d		= int
 * %f		= double
 * %B %/B	= bold on/off
 * %R %/R	= reverse on/off
 * %0-n% %/0-n%	= color on/off
 *
 */
void colprint(BBox * bbox, int y, int x, color * c, const char *fmt, ...)
{
	va_list			ap;
	unsigned int		i = 0;
	double			f = 0;
	string			output = "";
	bool			parse = false;
	bool			attr = false;
	attr_t			attrval = 0;
	char			buf[1024];
	string			colorstr;
	int			colorint;
	int			pair = 0;
	unsigned int		maxlen;		// max allowed characters printed on screen
	unsigned int		printlen = 0;	// num characters printed on screen

	assert(bbox);
	
	va_start(ap, fmt);

	/* Check if string is out of range, and cuts if necessary */
	if (x < 0)
	{
		if (strlen(fmt) < abs(x))
			return;

		fmt += abs(x);
		x = 0;
	}

	if (c != NULL)
		pair = c->pair();

	wmove(bbox->window, y, x);
	wattron(bbox->window, pair);

	maxlen = bbox->width() - x + 1;

	while(*fmt && printlen < maxlen)
	{
		if (*fmt == '%' && !parse)
		{
			if (*(fmt + 1) == '%')
			{
				fmt += 2;
				output = "%%";
				wprintw(bbox->window, _(output.c_str()));
				continue;
			}
			parse = true;
			attr = true;
			++fmt;
		}

		if (parse)
		{
			switch(*fmt)
			{
				case '/':
				/* Turn off attribute, SGML style */
					attr = false;
					break;
				case 'B':
					if (attr)
						wattron(bbox->window, A_BOLD);
					else
						wattroff(bbox->window, A_BOLD);
					parse = false;
					break;
				case 'R':
					if (attr)
						wattron(bbox->window, A_REVERSE);
					else
						wattroff(bbox->window, A_REVERSE);
					parse = false;
					break;
				case 'd':
					parse = false;
					i = va_arg(ap, int);
					sprintf(buf, "%d", i);
					wprintw(bbox->window, _(buf));
					printlen += strlen(buf);
					i = 0;
					break;
				case 'f':
					parse = false;
					f = va_arg(ap, double);
					sprintf(buf, "%f", f);
					wprintw(bbox->window, _(buf));
					printlen += strlen(buf);
					break;
				case 's':
					parse = false;
					output = va_arg(ap, const char *);
					if (output.size() >= (maxlen - printlen))
					{
						output = output.substr(0, (maxlen - printlen - 1));
					}
					sprintf(buf, "%s", output.c_str());
					wprintw(bbox->window, _(buf));
					printlen += strlen(buf);
					break;
				case 0:
					parse = false;
					continue;
				default:
					/* Use colors? */
					i = atoi(fmt);
					if (i >= 0)
					{
						if (attr)
						{
							wattroff(bbox->window, pair);
							wattron(bbox->window, i);
						}
						else
						{
							wattroff(bbox->window, i);
							wattron(bbox->window, pair);
						}

						/* Skip characters */
						colorint = static_cast<int>(i);
						colorstr = Pms::tostring(colorint);
						fmt += (colorstr.size());
					}
					parse = false;
					break;
			}
		}
		else
		{
			output = *fmt;
			wprintw(bbox->window, _(output.c_str()));
			++printlen;
		}
		++fmt;
	}

	va_end(ap);
	wattroff(bbox->window, pair);
}

